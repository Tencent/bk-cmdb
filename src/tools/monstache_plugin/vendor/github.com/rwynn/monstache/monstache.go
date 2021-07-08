// package main provides the monstache binary
package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"plugin"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/rwynn/monstache/pkg/oplog"

	"github.com/BurntSushi/toml"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/coreos/go-systemd/daemon"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/fsnotify/fsnotify"
	"github.com/olivere/elastic"
	aws "github.com/olivere/elastic/aws/v4"
	"github.com/robertkrimen/otto"
	_ "github.com/robertkrimen/otto/underscore"
	"github.com/rwynn/gtm"
	"github.com/rwynn/gtm/consistent"
	"github.com/rwynn/monstache/monstachemap"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
	mongoversion "go.mongodb.org/mongo-driver/version"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"gopkg.in/Graylog2/go-gelf.v2/gelf"
	"gopkg.in/natefinch/lumberjack.v2"
)

var infoLog = log.New(os.Stdout, "INFO ", log.Flags())
var warnLog = log.New(os.Stdout, "WARN ", log.Flags())
var statsLog = log.New(os.Stdout, "STATS ", log.Flags())
var traceLog = log.New(os.Stdout, "TRACE ", log.Flags())
var errorLog = log.New(os.Stderr, "ERROR ", log.Flags())

var initPlugin func(*monstachemap.InitPluginInput) error
var mapperPlugin func(*monstachemap.MapperPluginInput) (*monstachemap.MapperPluginOutput, error)
var filterPlugin func(*monstachemap.MapperPluginInput) (bool, error)
var processPlugin func(*monstachemap.ProcessPluginInput) error
var pipePlugin func(string, bool) ([]interface{}, error)
var mapEnvs = make(map[string]*executionEnv)
var filterEnvs = make(map[string]*executionEnv)
var pipeEnvs = make(map[string]*executionEnv)
var mapIndexTypes = make(map[string]*indexMapping)
var relates = make(map[string][]*relation)
var fileNamespaces = make(map[string]bool)
var patchNamespaces = make(map[string]bool)
var tmNamespaces = make(map[string]bool)
var routingNamespaces = make(map[string]bool)
var mux sync.Mutex

var chunksRegex = regexp.MustCompile("\\.chunks$")
var systemsRegex = regexp.MustCompile("system\\..+$")
var exitStatus = 0

const version = "6.7.5"
const mongoURLDefault string = "mongodb://localhost:27017"
const resumeNameDefault string = "default"
const elasticMaxConnsDefault int = 4
const elasticClientTimeoutDefault int = 0
const elasticMaxDocsDefault int = -1
const elasticMaxBytesDefault int = 8 * 1024 * 1024
const gtmChannelSizeDefault int = 512
const fileDownloadersDefault = 10
const relateThreadsDefault = 10
const relateBufferDefault = 1000
const postProcessorsDefault = 10
const redact = "REDACTED"
const configDatabaseNameDefault = "monstache"
const relateQueueOverloadMsg = "Relate queue is full. Skipping relate for %v.(%v) to keep pipeline healthy."

type awsCredentialStrategy int

const (
	awsCredentialStrategyStatic = iota
	awsCredentialStrategyFile
	awsCredentialStrategyEnv
	awsCredentialStrategyEndpoint
	awsCredentialStrategyChained
)

type deleteStrategy int

const (
	statelessDeleteStrategy deleteStrategy = iota
	statefulDeleteStrategy
	ignoreDeleteStrategy
)

type resumeStrategy int

const (
	timestampResumeStrategy resumeStrategy = iota
	tokenResumeStrategy
)

type buildInfo struct {
	Version      string
	VersionArray []int `bson:"versionArray"`
}

type stringargs []string

type indexClient struct {
	gtmCtx          *gtm.OpCtxMulti
	config          *configOptions
	mongo           *mongo.Client
	mongoConfig     *mongo.Client
	bulk            *elastic.BulkProcessor
	bulkStats       *elastic.BulkProcessor
	client          *elastic.Client
	hsc             *httpServerCtx
	fileWg          *sync.WaitGroup
	indexWg         *sync.WaitGroup
	processWg       *sync.WaitGroup
	relateWg        *sync.WaitGroup
	opsConsumed     chan bool
	closeC          chan bool
	doneC           chan int
	enabled         bool
	lastTs          primitive.Timestamp
	lastTsSaved     primitive.Timestamp
	tokens          bson.M
	indexC          chan *gtm.Op
	processC        chan *gtm.Op
	fileC           chan *gtm.Op
	relateC         chan *gtm.Op
	filter          gtm.OpFilter
	statusReqC      chan *statusRequest
	sigH            *sigHandler
	oplogTsResolver oplog.TimestampResolver
}

type sigHandler struct {
	clientStartedC chan *indexClient
}

type awsConnect struct {
	Strategy            awsCredentialStrategy
	AccessKey           string `toml:"access-key"`
	SecretKey           string `toml:"secret-key"`
	Region              string
	Profile             string
	CredentialsFile     string `toml:"credentials-file"`
	CredentialsWatchDir string `toml:"credentials-watch-dir"`
	WatchCredentials    bool   `toml:"watch-credentials"`
	ForceExpire         string `toml:"force-expire"`
	creds               *credentials.Credentials
}

type executionEnv struct {
	VM     *otto.Otto
	Script string
	lock   *sync.Mutex
}

type javascript struct {
	Namespace string
	Script    string
	Path      string
	Routing   bool
}

type relation struct {
	Namespace      string
	WithNamespace  string `toml:"with-namespace"`
	SrcField       string `toml:"src-field"`
	MatchField     string `toml:"match-field"`
	DotNotation    bool   `toml:"dot-notation"`
	KeepSrc        bool   `toml:"keep-src"`
	MaxDepth       int    `toml:"max-depth"`
	MatchFieldType string `toml:"match-field-type"`
	db             string
	col            string
}

type indexMapping struct {
	Namespace string
	Index     string
	Pipeline  string
}

type findConf struct {
	vm            *otto.Otto
	ns            string
	name          string
	client        *mongo.Client
	byID          bool
	multi         bool
	pipe          bool
	pipeAllowDisk bool
}

type findCall struct {
	config *findConf
	client *mongo.Client
	query  interface{}
	db     string
	col    string
	limit  int
	sort   map[string]int
	sel    map[string]int
}

type logRotate struct {
	MaxSize    int  `toml:"max-size"`
	MaxAge     int  `toml:"max-age"`
	MaxBackups int  `toml:"max-backups"`
	LocalTime  bool `toml:"localtime"`
	Compress   bool `toml:"compress"`
}

type logFiles struct {
	Info  string
	Warn  string
	Error string
	Trace string
	Stats string
}

type indexingMeta struct {
	Routing         string
	Index           string
	Type            string
	Parent          string
	Version         int64
	VersionType     string
	Pipeline        string
	RetryOnConflict int
	Skip            bool
	ID              string
}

type gtmSettings struct {
	ChannelSize    int    `toml:"channel-size"`
	BufferSize     int    `toml:"buffer-size"`
	BufferDuration string `toml:"buffer-duration"`
	MaxAwaitTime   string `toml:"max-await-time"`
}

type elasticPKIAuth struct {
	CertFile string `toml:"cert-file"`
	KeyFile  string `toml:"key-file"`
}

type httpServerCtx struct {
	httpServer *http.Server
	bulk       *elastic.BulkProcessor
	config     *configOptions
	shutdown   bool
	started    time.Time
	statusReqC chan *statusRequest
}

type instanceStatus struct {
	Enabled      bool                `json:"enabled"`
	Pid          int                 `json:"pid"`
	Hostname     string              `json:"hostname"`
	ClusterName  string              `json:"cluster"`
	ResumeName   string              `json:"resumeName"`
	LastTs       primitive.Timestamp `json:"lastTs"`
	LastTsFormat string              `json:"lastTsFormat,omitempty"`
}

type statusResponse struct {
	enabled bool
	lastTs  primitive.Timestamp
}

type statusRequest struct {
	responseC chan *statusResponse
}

type configOptions struct {
	EnableTemplate              bool
	EnvDelimiter                string
	MongoURL                    string         `toml:"mongo-url"`
	MongoConfigURL              string         `toml:"mongo-config-url"`
	MongoOpLogDatabaseName      string         `toml:"mongo-oplog-database-name"`
	MongoOpLogCollectionName    string         `toml:"mongo-oplog-collection-name"`
	GtmSettings                 gtmSettings    `toml:"gtm-settings"`
	AWSConnect                  awsConnect     `toml:"aws-connect"`
	LogRotate                   logRotate      `toml:"log-rotate"`
	Logs                        logFiles       `toml:"logs"`
	GraylogAddr                 string         `toml:"graylog-addr"`
	ElasticUrls                 stringargs     `toml:"elasticsearch-urls"`
	ElasticUser                 string         `toml:"elasticsearch-user"`
	ElasticPassword             string         `toml:"elasticsearch-password"`
	ElasticPemFile              string         `toml:"elasticsearch-pem-file"`
	ElasticValidatePemFile      bool           `toml:"elasticsearch-validate-pem-file"`
	ElasticVersion              string         `toml:"elasticsearch-version"`
	ElasticHealth0              int            `toml:"elasticsearch-healthcheck-timeout-startup"`
	ElasticHealth1              int            `toml:"elasticsearch-healthcheck-timeout"`
	ElasticPKIAuth              elasticPKIAuth `toml:"elasticsearch-pki-auth"`
	ResumeName                  string         `toml:"resume-name"`
	NsRegex                     string         `toml:"namespace-regex"`
	NsDropRegex                 string         `toml:"namespace-drop-regex"`
	NsExcludeRegex              string         `toml:"namespace-exclude-regex"`
	NsDropExcludeRegex          string         `toml:"namespace-drop-exclude-regex"`
	ClusterName                 string         `toml:"cluster-name"`
	Print                       bool           `toml:"print-config"`
	Version                     bool
	Pprof                       bool
	EnableOplog                 bool `toml:"enable-oplog"`
	DisableChangeEvents         bool `toml:"disable-change-events"`
	EnableEasyJSON              bool `toml:"enable-easy-json"`
	Stats                       bool
	IndexStats                  bool   `toml:"index-stats"`
	StatsDuration               string `toml:"stats-duration"`
	StatsIndexFormat            string `toml:"stats-index-format"`
	Gzip                        bool
	Verbose                     bool
	Resume                      bool
	ResumeStrategy              resumeStrategy `toml:"resume-strategy"`
	ResumeWriteUnsafe           bool           `toml:"resume-write-unsafe"`
	ResumeFromTimestamp         int64          `toml:"resume-from-timestamp"`
	ResumeFromEarliestTimestamp bool           `toml:"resume-from-earliest-timestamp"`
	Replay                      bool
	DroppedDatabases            bool   `toml:"dropped-databases"`
	DroppedCollections          bool   `toml:"dropped-collections"`
	IndexFiles                  bool   `toml:"index-files"`
	IndexAsUpdate               bool   `toml:"index-as-update"`
	FileHighlighting            bool   `toml:"file-highlighting"`
	DisableFilePipelinePut      bool   `toml:"disable-file-pipeline-put"`
	EnablePatches               bool   `toml:"enable-patches"`
	FailFast                    bool   `toml:"fail-fast"`
	IndexOplogTime              bool   `toml:"index-oplog-time"`
	OplogTsFieldName            string `toml:"oplog-ts-field-name"`
	OplogDateFieldName          string `toml:"oplog-date-field-name"`
	OplogDateFieldFormat        string `toml:"oplog-date-field-format"`
	ExitAfterDirectReads        bool   `toml:"exit-after-direct-reads"`
	MergePatchAttr              string `toml:"merge-patch-attribute"`
	ElasticMaxConns             int    `toml:"elasticsearch-max-conns"`
	ElasticRetry                bool   `toml:"elasticsearch-retry"`
	ElasticMaxDocs              int    `toml:"elasticsearch-max-docs"`
	ElasticMaxBytes             int    `toml:"elasticsearch-max-bytes"`
	ElasticMaxSeconds           int    `toml:"elasticsearch-max-seconds"`
	ElasticClientTimeout        int    `toml:"elasticsearch-client-timeout"`
	ElasticMajorVersion         int
	ElasticMinorVersion         int
	MaxFileSize                 int64 `toml:"max-file-size"`
	ConfigFile                  string
	Script                      []javascript
	Filter                      []javascript
	Pipeline                    []javascript
	Mapping                     []indexMapping
	Relate                      []relation
	FileNamespaces              stringargs `toml:"file-namespaces"`
	PatchNamespaces             stringargs `toml:"patch-namespaces"`
	Workers                     stringargs
	Worker                      string
	ChangeStreamNs              stringargs     `toml:"change-stream-namespaces"`
	DirectReadNs                stringargs     `toml:"direct-read-namespaces"`
	DirectReadSplitMax          int            `toml:"direct-read-split-max"`
	DirectReadConcur            int            `toml:"direct-read-concur"`
	DirectReadNoTimeout         bool           `toml:"direct-read-no-timeout"`
	DirectReadBounded           bool           `toml:"direct-read-bounded"`
	DirectReadStateful          bool           `toml:"direct-read-stateful"`
	DirectReadExcludeRegex      string         `toml:"direct-read-dynamic-exclude-regex"`
	DirectReadIncludeRegex      string         `toml:"direct-read-dynamic-include-regex"`
	MapperPluginPath            string         `toml:"mapper-plugin-path"`
	EnableHTTPServer            bool           `toml:"enable-http-server"`
	HTTPServerAddr              string         `toml:"http-server-addr"`
	TimeMachineNamespaces       stringargs     `toml:"time-machine-namespaces"`
	TimeMachineIndexPrefix      string         `toml:"time-machine-index-prefix"`
	TimeMachineIndexSuffix      string         `toml:"time-machine-index-suffix"`
	TimeMachineDirectReads      bool           `toml:"time-machine-direct-reads"`
	PipeAllowDisk               bool           `toml:"pipe-allow-disk"`
	RoutingNamespaces           stringargs     `toml:"routing-namespaces"`
	DeleteStrategy              deleteStrategy `toml:"delete-strategy"`
	DeleteIndexPattern          string         `toml:"delete-index-pattern"`
	ConfigDatabaseName          string         `toml:"config-database-name"`
	FileDownloaders             int            `toml:"file-downloaders"`
	RelateThreads               int            `toml:"relate-threads"`
	RelateBuffer                int            `toml:"relate-buffer"`
	PostProcessors              int            `toml:"post-processors"`
	PruneInvalidJSON            bool           `toml:"prune-invalid-json"`
	Debug                       bool
	mongoClientOptions          *options.ClientOptions
}

func (eca elasticPKIAuth) enabled() bool {
	return eca.CertFile != "" || eca.KeyFile != ""
}

func (eca elasticPKIAuth) validate() error {
	if eca.CertFile != "" && eca.KeyFile == "" {
		return errors.New("Elasticsearch client auth key file is empty")
	}
	if eca.CertFile == "" && eca.KeyFile != "" {
		return errors.New("Elasticsearch client auth cert file is empty")
	}
	return nil
}

func (rel *relation) IsIdentity() bool {
	if rel.SrcField == "_id" && rel.MatchField == "_id" {
		return true
	}
	return false
}

func (l *logFiles) enabled() bool {
	return l.Info != "" || l.Warn != "" || l.Error != "" || l.Trace != "" || l.Stats != ""
}

func (ac *awsConnect) validate() error {
	if ac.Strategy == awsCredentialStrategyStatic {
		if ac.AccessKey == "" && ac.SecretKey == "" {
			return nil
		} else if ac.AccessKey != "" && ac.SecretKey != "" {
			return nil
		}
		return errors.New("AWS connect settings must include both access-key and secret-key")
	}
	return nil
}

func (ac *awsConnect) enabled() bool {
	if ac.Strategy == awsCredentialStrategyStatic {
		return ac.AccessKey != "" || ac.SecretKey != ""
	}
	return true
}

func (ac *awsConnect) forceExpireCreds() bool {
	return ac.enabled() && ac.ForceExpire != "" && ac.creds != nil
}

func (ac *awsConnect) watchCreds() bool {
	if ac.enabled() && ac.creds != nil && ac.WatchCredentials {
		return ac.Strategy == awsCredentialStrategyFile || ac.Strategy == awsCredentialStrategyChained
	}
	return false
}

func (ac *awsConnect) watchFilePath() string {
	if ac.CredentialsWatchDir != "" {
		return ac.CredentialsWatchDir
	}
	var homeDir string
	if runtime.GOOS == "windows" { // Windows
		homeDir = os.Getenv("USERPROFILE")
	} else {
		homeDir = os.Getenv("HOME")
	}
	return filepath.Join(homeDir, ".aws")
}

func (arg *deleteStrategy) String() string {
	return fmt.Sprintf("%d", *arg)
}

func (arg *deleteStrategy) Set(value string) (err error) {
	var i int
	if i, err = strconv.Atoi(value); err != nil {
		return
	}
	ds := deleteStrategy(i)
	*arg = ds
	return
}

func (arg *resumeStrategy) String() string {
	return fmt.Sprintf("%d", *arg)
}

func (arg *resumeStrategy) Set(value string) (err error) {
	var i int
	if i, err = strconv.Atoi(value); err != nil {
		return
	}
	rs := resumeStrategy(i)
	*arg = rs
	return
}

func (args *stringargs) String() string {
	return fmt.Sprintf("%s", *args)
}

func (args *stringargs) Set(value string) error {
	*args = append(*args, value)
	return nil
}

func (config *configOptions) readShards() bool {
	return len(config.ChangeStreamNs) == 0 && config.MongoConfigURL != ""
}

func (config *configOptions) dynamicDirectReadList() bool {
	return len(config.DirectReadNs) == 1 && config.DirectReadNs[0] == ""
}

func (config *configOptions) dynamicChangeStreamList() bool {
	return len(config.ChangeStreamNs) == 1 && config.ChangeStreamNs[0] == ""
}

func (config *configOptions) ignoreDatabaseForDirectReads(db string) bool {
	return db == "local" || db == "admin" || db == "config" || db == config.ConfigDatabaseName
}

func (config *configOptions) ignoreCollectionForDirectReads(col string) bool {
	return strings.HasPrefix(col, "system.")
}

func (config *configOptions) ignoreDatabaseForChangeStreamReads(db string) bool {
	return config.ignoreDatabaseForDirectReads(db)
}

func (config *configOptions) ignoreCollectionForChangeStreamReads(col string) bool {
	return config.ignoreCollectionForDirectReads(col)
}

func afterBulk(executionID int64, requests []elastic.BulkableRequest, response *elastic.BulkResponse, err error) {
	if response == nil || !response.Errors {
		return
	}
	if failed := response.Failed(); failed != nil {
		for _, item := range failed {
			if item.Status == 409 {
				// ignore version conflict since this simply means the doc
				// is already in the index
				continue
			}
			json, err := json.Marshal(item)
			if err != nil {
				errorLog.Printf("Unable to marshal bulk response item: %s", err)
			} else {
				errorLog.Printf("Bulk response item: %s", string(json))
			}
		}
	}
}

func (config *configOptions) parseElasticsearchVersion(number string) (err error) {
	if number == "" {
		err = errors.New("Elasticsearch version cannot be blank")
	} else {
		versionParts := strings.Split(number, ".")
		var majorVersion, minorVersion int
		majorVersion, err = strconv.Atoi(versionParts[0])
		if err == nil {
			config.ElasticMajorVersion = majorVersion
			if majorVersion == 0 {
				err = errors.New("Invalid Elasticsearch major version 0")
			}
		}
		if len(versionParts) > 1 {
			minorVersion, err = strconv.Atoi(versionParts[1])
			if err == nil {
				config.ElasticMinorVersion = minorVersion
			}
		}
	}
	return
}

func (config *configOptions) newBulkProcessor(client *elastic.Client) (bulk *elastic.BulkProcessor, err error) {
	bulkService := client.BulkProcessor().Name("monstache")
	bulkService.Workers(config.ElasticMaxConns)
	bulkService.Stats(config.Stats)
	bulkService.BulkActions(config.ElasticMaxDocs)
	bulkService.BulkSize(config.ElasticMaxBytes)
	if config.ElasticRetry == false {
		bulkService.Backoff(&elastic.StopBackoff{})
	}
	bulkService.After(afterBulk)
	bulkService.FlushInterval(time.Duration(config.ElasticMaxSeconds) * time.Second)
	return bulkService.Do(context.Background())
}

func (config *configOptions) newStatsBulkProcessor(client *elastic.Client) (bulk *elastic.BulkProcessor, err error) {
	bulkService := client.BulkProcessor().Name("monstache-stats")
	bulkService.Workers(1)
	bulkService.Stats(false)
	bulkService.BulkActions(-1)
	bulkService.BulkSize(-1)
	bulkService.After(afterBulk)
	bulkService.FlushInterval(time.Duration(5) * time.Second)
	return bulkService.Do(context.Background())
}

func (config *configOptions) needsSecureScheme() bool {
	if len(config.ElasticUrls) > 0 {
		for _, url := range config.ElasticUrls {
			if strings.HasPrefix(url, "https") {
				return true
			}
		}
	}
	return false

}

func (config *configOptions) newElasticClient() (client *elastic.Client, err error) {
	var clientOptions []elastic.ClientOptionFunc
	var httpClient *http.Client
	clientOptions = append(clientOptions, elastic.SetSniff(false))
	if config.needsSecureScheme() {
		clientOptions = append(clientOptions, elastic.SetScheme("https"))
	}
	if len(config.ElasticUrls) > 0 {
		clientOptions = append(clientOptions, elastic.SetURL(config.ElasticUrls...))
	} else {
		config.ElasticUrls = append(config.ElasticUrls, elastic.DefaultURL)
	}
	if config.Verbose {
		clientOptions = append(clientOptions, elastic.SetTraceLog(traceLog))
		clientOptions = append(clientOptions, elastic.SetErrorLog(errorLog))
	}
	if config.ElasticUser != "" {
		clientOptions = append(clientOptions, elastic.SetBasicAuth(config.ElasticUser, config.ElasticPassword))
	}
	if config.ElasticRetry {
		d1, d2 := time.Duration(50)*time.Millisecond, time.Duration(20)*time.Second
		retrier := elastic.NewBackoffRetrier(elastic.NewExponentialBackoff(d1, d2))
		clientOptions = append(clientOptions, elastic.SetRetrier(retrier))
	}
	httpClient, err = config.NewHTTPClient()
	if err != nil {
		return client, err
	}
	clientOptions = append(clientOptions, elastic.SetHttpClient(httpClient))
	clientOptions = append(clientOptions,
		elastic.SetHealthcheckTimeoutStartup(time.Duration(config.ElasticHealth0)*time.Second))
	clientOptions = append(clientOptions,
		elastic.SetHealthcheckTimeout(time.Duration(config.ElasticHealth1)*time.Second))
	return elastic.NewClient(clientOptions...)
}

func (config *configOptions) testElasticsearchConn(client *elastic.Client) (err error) {
	var number string
	url := config.ElasticUrls[0]
	number, err = client.ElasticsearchVersion(url)
	if err == nil {
		infoLog.Printf("Successfully connected to Elasticsearch version %s", number)
		err = config.parseElasticsearchVersion(number)
	}
	return
}

func (ic *indexClient) deleteIndexes(db string) (err error) {
	var indices = []string{strings.ToLower(db + ".*")}
	for ns, m := range mapIndexTypes {
		dbCol := strings.SplitN(ns, ".", 2)
		if dbCol[0] == db && m.Index != "" {
			index := strings.ToLower(m.Index)
			for _, cur := range indices {
				if cur == index {
					index = ""
					break
				}
			}
			if index != "" {
				indices = append(indices, index)
			}
		}
	}
	_, err = ic.client.DeleteIndex(indices...).Do(context.Background())
	return
}

func (ic *indexClient) deleteIndex(namespace string) (err error) {
	ctx := context.Background()
	index := strings.ToLower(namespace)
	if m := mapIndexTypes[namespace]; m != nil {
		if m.Index != "" {
			index = strings.ToLower(m.Index)
		}
	}
	_, err = ic.client.DeleteIndex(index).Do(ctx)
	return err
}

func (ic *indexClient) ensureFileMapping() (err error) {
	config := ic.config
	if config.DisableFilePipelinePut {
		return nil
	}
	ctx := context.Background()
	pipeline := map[string]interface{}{
		"description": "Extract file information",
		"processors": [1]map[string]interface{}{
			{
				"attachment": map[string]interface{}{
					"field": "file",
				},
			},
		},
	}
	_, err = ic.client.IngestPutPipeline("attachment").BodyJson(pipeline).Do(ctx)
	return err
}

func (ic *indexClient) defaultIndexMapping(op *gtm.Op) *indexMapping {
	return &indexMapping{
		Namespace: op.Namespace,
		Index:     strings.ToLower(op.Namespace),
	}
}

func (ic *indexClient) mapIndex(op *gtm.Op) *indexMapping {
	mapping := ic.defaultIndexMapping(op)
	if m := mapIndexTypes[op.Namespace]; m != nil {
		if m.Index != "" {
			mapping.Index = m.Index
		}
		if m.Pipeline != "" {
			mapping.Pipeline = m.Pipeline
		}
	}
	return mapping
}

func opIDToString(op *gtm.Op) string {
	var opIDStr string
	switch id := op.Id.(type) {
	case primitive.ObjectID:
		opIDStr = id.Hex()
	case primitive.Binary:
		opIDStr = monstachemap.EncodeBinData(monstachemap.Binary{id})
	case float64:
		intID := int(id)
		if id == float64(intID) {
			opIDStr = fmt.Sprintf("%v", intID)
		} else {
			opIDStr = fmt.Sprintf("%v", op.Id)
		}
	case float32:
		intID := int(id)
		if id == float32(intID) {
			opIDStr = fmt.Sprintf("%v", intID)
		} else {
			opIDStr = fmt.Sprintf("%v", op.Id)
		}
	default:
		opIDStr = fmt.Sprintf("%v", op.Id)
	}
	return opIDStr
}

func convertSliceJavascript(a []interface{}) []interface{} {
	var avs []interface{}
	for _, av := range a {
		var avc interface{}
		switch achild := av.(type) {
		case map[string]interface{}:
			avc = convertMapJavascript(achild)
		case []interface{}:
			avc = convertSliceJavascript(achild)
		case primitive.ObjectID:
			avc = achild.Hex()
		default:
			avc = av
		}
		avs = append(avs, avc)
	}
	return avs
}

func convertMapJavascript(e map[string]interface{}) map[string]interface{} {
	o := make(map[string]interface{})
	for k, v := range e {
		switch child := v.(type) {
		case map[string]interface{}:
			o[k] = convertMapJavascript(child)
		case []interface{}:
			o[k] = convertSliceJavascript(child)
		case primitive.ObjectID:
			o[k] = child.Hex()
		default:
			o[k] = v
		}
	}
	return o
}

func fixSlicePruneInvalidJSON(id string, key string, a []interface{}) []interface{} {
	var avs []interface{}
	for _, av := range a {
		var avc interface{}
		switch achild := av.(type) {
		case map[string]interface{}:
			avc = fixPruneInvalidJSON(id, achild)
		case []interface{}:
			avc = fixSlicePruneInvalidJSON(id, key, achild)
		case time.Time:
			year := achild.Year()
			if year < 0 || year > 9999 {
				// year outside of valid range
				warnLog.Printf("Dropping key %s element: invalid time.Time value: %s for document _id: %s", key, achild, id)
				continue
			} else {
				avc = av
			}
		case float64:
			if math.IsNaN(achild) {
				// causes an error in the json serializer
				warnLog.Printf("Dropping key %s element: invalid float64 value: %v for document _id: %s", key, achild, id)
				continue
			} else if math.IsInf(achild, 0) {
				// causes an error in the json serializer
				warnLog.Printf("Dropping key %s element: invalid float64 value: %v for document _id: %s", key, achild, id)
				continue
			} else {
				avc = av
			}
		default:
			avc = av
		}
		avs = append(avs, avc)
	}
	return avs
}

func fixPruneInvalidJSON(id string, e map[string]interface{}) map[string]interface{} {
	o := make(map[string]interface{})
	for k, v := range e {
		switch child := v.(type) {
		case map[string]interface{}:
			o[k] = fixPruneInvalidJSON(id, child)
		case []interface{}:
			o[k] = fixSlicePruneInvalidJSON(id, k, child)
		case time.Time:
			year := child.Year()
			if year < 0 || year > 9999 {
				// year outside of valid range
				warnLog.Printf("Dropping key %s: invalid time.Time value: %s for document _id: %s", k, child, id)
				continue
			} else {
				o[k] = v
			}
		case float64:
			if math.IsNaN(child) {
				// causes an error in the json serializer
				warnLog.Printf("Dropping key %s: invalid float64 value: %v for document _id: %s", k, child, id)
				continue
			} else if math.IsInf(child, 0) {
				// causes an error in the json serializer
				warnLog.Printf("Dropping key %s: invalid float64 value: %v for document _id: %s", k, child, id)
				continue
			} else {
				o[k] = v
			}
		default:
			o[k] = v
		}
	}
	return o
}

func deepExportValue(a interface{}) (b interface{}) {
	switch t := a.(type) {
	case otto.Value:
		ex, err := t.Export()
		if t.Class() == "Date" {
			ex, err = time.Parse("Mon, 2 Jan 2006 15:04:05 MST", t.String())
		}
		if err == nil {
			b = deepExportValue(ex)
		} else {
			errorLog.Printf("Error exporting from javascript: %s", err)
		}
	case map[string]interface{}:
		b = deepExportMap(t)
	case []map[string]interface{}:
		b = deepExportMapSlice(t)
	case []interface{}:
		b = deepExportSlice(t)
	default:
		b = a
	}
	return
}

func deepExportMapSlice(a []map[string]interface{}) []interface{} {
	var avs []interface{}
	for _, av := range a {
		avs = append(avs, deepExportMap(av))
	}
	return avs
}

func deepExportSlice(a []interface{}) []interface{} {
	var avs []interface{}
	for _, av := range a {
		avs = append(avs, deepExportValue(av))
	}
	return avs
}

func deepExportMap(e map[string]interface{}) map[string]interface{} {
	o := make(map[string]interface{})
	for k, v := range e {
		o[k] = deepExportValue(v)
	}
	return o
}

func (ic *indexClient) mapDataJavascript(op *gtm.Op) error {
	names := []string{"", op.Namespace}
	for _, name := range names {
		env := mapEnvs[name]
		if env == nil {
			continue
		}
		env.lock.Lock()
		defer env.lock.Unlock()
		arg := convertMapJavascript(op.Data)
		arg2 := op.Namespace
		arg3 := convertMapJavascript(op.UpdateDescription)
		val, err := env.VM.Call("module.exports", arg, arg, arg2, arg3)
		if err != nil {
			return err
		}
		if strings.ToLower(val.Class()) == "object" {
			data, err := val.Export()
			if err != nil {
				return err
			} else if data == val {
				return errors.New("Exported function must return an object")
			} else {
				dm := data.(map[string]interface{})
				op.Data = deepExportMap(dm)
			}
		} else {
			indexed, err := val.ToBoolean()
			if err != nil {
				return err
			} else if !indexed {
				op.Data = nil
				break
			}
		}
	}
	return nil
}

func (ic *indexClient) mapDataGolang(op *gtm.Op) error {
	input := &monstachemap.MapperPluginInput{
		Document:             op.Data,
		Namespace:            op.Namespace,
		Database:             op.GetDatabase(),
		Collection:           op.GetCollection(),
		Operation:            op.Operation,
		MongoClient:          ic.mongo,
		ElasticClient:        ic.client,
		ElasticBulkProcessor: ic.bulk,
		UpdateDescription:    op.UpdateDescription,
	}
	output, err := mapperPlugin(input)
	if err != nil {
		return err
	}
	if output == nil {
		return nil
	}
	if output.Drop {
		op.Data = nil
	} else {
		if output.Skip {
			op.Data = map[string]interface{}{}
		} else if output.Passthrough == false {
			if output.Document == nil {
				return errors.New("Map function must return a non-nil document")
			}
			op.Data = output.Document
		}
		meta := make(map[string]interface{})
		if output.Skip {
			meta["skip"] = true
		}
		if output.Index != "" {
			meta["index"] = output.Index
		}
		if output.ID != "" {
			meta["id"] = output.ID
		}
		if output.Type != "" {
			meta["type"] = output.Type
		}
		if output.Routing != "" {
			meta["routing"] = output.Routing
		}
		if output.Parent != "" {
			meta["parent"] = output.Parent
		}
		if output.Version != 0 {
			meta["version"] = output.Version
		}
		if output.VersionType != "" {
			meta["versionType"] = output.VersionType
		}
		if output.Pipeline != "" {
			meta["pipeline"] = output.Pipeline
		}
		if output.RetryOnConflict != 0 {
			meta["retryOnConflict"] = output.RetryOnConflict
		}
		if len(meta) > 0 {
			op.Data["_meta_monstache"] = meta
		}
	}
	return nil
}

func (ic *indexClient) mapData(op *gtm.Op) error {
	if mapperPlugin != nil {
		return ic.mapDataGolang(op)
	}
	return ic.mapDataJavascript(op)
}

func extractData(srcField string, data map[string]interface{}) (result interface{}, err error) {
	var cur = data
	fields := strings.Split(srcField, ".")
	flen := len(fields)
	for i, field := range fields {
		if i+1 == flen {
			result = cur[field]
		} else {
			if next, ok := cur[field].(map[string]interface{}); ok {
				cur = next
			} else {
				break
			}
		}
	}
	if result == nil {
		var detail interface{}
		b, e := json.Marshal(data)
		if e == nil {
			detail = string(b)
		} else {
			detail = err
		}
		err = fmt.Errorf("Source field %s not found in document: %s", srcField, detail)
	}
	return
}

func buildSelector(matchField string, data interface{}) bson.M {
	sel := bson.M{}
	var cur bson.M = sel
	fields := strings.Split(matchField, ".")
	flen := len(fields)
	for i, field := range fields {
		if i+1 == flen {
			cur[field] = data
		} else {
			next := bson.M{}
			cur[field] = next
			cur = next
		}
	}
	return sel
}

func convertSrcDataToString(srcData interface{}) (value string) {
	switch v := srcData.(type) {
	case primitive.ObjectID:
		value = v.Hex()
	default:
		value = fmt.Sprintf("%v", v)
	}
	return
}

func convertSrcDataToObjectID(srcData interface{}) (objectID primitive.ObjectID, err error) {
	value := fmt.Sprintf("%v", srcData)
	objectID, err = primitive.ObjectIDFromHex(value)
	return
}

func convertSrcDataToInt(srcData interface{}) (val int64, err error) {
	switch v := srcData.(type) {
	case int:
		return int64(v), nil
	case int64:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case float32:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	case bool:
		vals := map[bool]int64{false: 0, true: 1}
		return vals[v], nil
	case primitive.Decimal128:
		return strconv.ParseInt(v.String(), 10, 64)
	}
	return 0, fmt.Errorf("Failed to convert match field of type %T to int64", srcData)
}

func convertSrcDataToDecimal(srcData interface{}) (decimal primitive.Decimal128, err error) {
	switch v := srcData.(type) {
	case int, int64, int32, float64, float32:
		return primitive.ParseDecimal128(fmt.Sprintf("%v", v))
	case string:
		return primitive.ParseDecimal128(v)
	case primitive.Decimal128:
		return v, nil
	}
	return decimal, fmt.Errorf("Failed to convert match field of type %T to decimal", srcData)
}

func convertSrcDataToMatchFieldType(srcData interface{}, matchFieldType string) (result interface{}, err error) {
	if matchFieldType == "objectId" {
		result, err = convertSrcDataToObjectID(srcData)
	} else if matchFieldType == "string" {
		result = convertSrcDataToString(srcData)
	} else if matchFieldType == "int" || matchFieldType == "long" {
		result, err = convertSrcDataToInt(srcData)
	} else if matchFieldType == "decimal" {
		result, err = convertSrcDataToDecimal(srcData)
	}
	return
}

func (ic *indexClient) processRelated(root *gtm.Op) (err error) {
	var q []*gtm.Op
	batch := []*gtm.Op{root}
	depth := 1
	for len(batch) > 0 {
		for _, e := range batch {
			op := e
			if op.Data == nil {
				continue
			}
			rs := relates[op.Namespace]
			if len(rs) == 0 {
				continue
			}
			for _, r := range rs {
				if r.MaxDepth > 0 && r.MaxDepth < depth {
					continue
				}
				if op.IsDelete() && r.IsIdentity() {
					rop := &gtm.Op{
						Id:        op.Id,
						Operation: op.Operation,
						Namespace: r.WithNamespace,
						Source:    op.Source,
						Timestamp: op.Timestamp,
						Data:      op.Data,
					}
					ic.doDelete(rop)
					q = append(q, rop)
					continue
				}
				var srcData interface{}
				if srcData, err = extractData(r.SrcField, op.Data); err != nil {
					ic.processErr(err)
					continue
				}

				if r.MatchFieldType != "" {
					if srcData, err = convertSrcDataToMatchFieldType(srcData, r.MatchFieldType); err != nil {
						ic.processErr(err)
						continue
					}
				}

				opts := &options.FindOptions{}
				if ic.config.DirectReadNoTimeout {
					opts.SetNoCursorTimeout(true)
				}
				col := ic.mongo.Database(r.db).Collection(r.col)
				var sel bson.M
				if r.DotNotation {
					sel = bson.M{r.MatchField: srcData}
				} else {
					sel = buildSelector(r.MatchField, srcData)
				}
				cursor, err := col.Find(context.Background(), sel, opts)

				doc := make(map[string]interface{})
				for cursor.Next(context.Background()) {
					if err = cursor.Decode(&doc); err != nil {
						ic.processErr(err)
						continue
					}
					now := time.Now().UTC()
					tstamp := primitive.Timestamp{
						T: uint32(now.Unix()),
						I: uint32(now.Nanosecond()),
					}
					rop := &gtm.Op{
						Id:                doc["_id"],
						Data:              doc,
						Operation:         root.Operation,
						Namespace:         r.WithNamespace,
						Source:            gtm.DirectQuerySource,
						Timestamp:         tstamp,
						UpdateDescription: root.UpdateDescription,
					}
					doc = make(map[string]interface{})
					if ic.filter != nil && !ic.filter(rop) {
						continue
					}
					if processPlugin != nil {
						pop := &gtm.Op{
							Id:                rop.Id,
							Operation:         rop.Operation,
							Namespace:         rop.Namespace,
							Source:            rop.Source,
							Timestamp:         rop.Timestamp,
							UpdateDescription: rop.UpdateDescription,
						}
						var data []byte
						data, err = bson.Marshal(rop.Data)
						if err == nil {
							var m map[string]interface{}
							err = bson.Unmarshal(data, &m)
							if err == nil {
								pop.Data = m
							}
						}
						ic.processC <- pop
					}
					skip := false
					if rs2 := relates[rop.Namespace]; len(rs2) != 0 {
						skip = true
						visit := false
						for _, r2 := range rs2 {
							if r2.KeepSrc {
								skip = false
							}
							if r2.MaxDepth < 1 || r2.MaxDepth >= (depth+1) {
								visit = true
							}
						}
						if visit {
							q = append(q, rop)
						}
					}
					if !skip {
						if ic.hasFileContent(rop) {
							ic.fileC <- rop
						} else {
							ic.indexC <- rop
						}
					}
				}
				cursor.Close(context.Background())
			}
		}
		depth++
		batch = q
		q = nil
	}
	return
}

func (ic *indexClient) prepareDataForIndexing(op *gtm.Op) {
	config := ic.config
	data := op.Data
	if config.IndexOplogTime {
		secs := op.Timestamp.T
		t := time.Unix(int64(secs), 0).UTC()
		data[config.OplogTsFieldName] = op.Timestamp
		data[config.OplogDateFieldName] = t.Format(config.OplogDateFieldFormat)
	}
	delete(data, "_id")
	delete(data, "_meta_monstache")
	if config.PruneInvalidJSON {
		op.Data = fixPruneInvalidJSON(opIDToString(op), data)
	}
	op.Data = monstachemap.ConvertMapForJSON(op.Data)
}

func parseIndexMeta(op *gtm.Op) (meta *indexingMeta) {
	meta = &indexingMeta{
		Version:     tsVersion(op.Timestamp),
		VersionType: "external",
	}
	if m, ok := op.Data["_meta_monstache"]; ok {
		switch m.(type) {
		case map[string]interface{}:
			metaAttrs := m.(map[string]interface{})
			meta.load(metaAttrs)
		case otto.Value:
			ex, err := m.(otto.Value).Export()
			if err == nil && ex != m {
				switch ex.(type) {
				case map[string]interface{}:
					metaAttrs := ex.(map[string]interface{})
					meta.load(metaAttrs)
				default:
					errorLog.Println("Invalid indexing metadata")
				}
			}
		default:
			errorLog.Println("Invalid indexing metadata")
		}
	}
	return meta
}

func (ic *indexClient) addFileContent(op *gtm.Op) (err error) {
	op.Data["file"] = ""
	var gridByteBuffer bytes.Buffer
	db, bucketName :=
		ic.mongo.Database(op.GetDatabase()),
		strings.SplitN(op.GetCollection(), ".", 2)[0]
	encoder := base64.NewEncoder(base64.StdEncoding, &gridByteBuffer)
	opts := &options.BucketOptions{}
	opts.SetName(bucketName)
	var bucket *gridfs.Bucket
	bucket, err = gridfs.NewBucket(db, opts)
	if err != nil {
		return
	}
	var size int64
	if size, err = bucket.DownloadToStream(op.Id, encoder); err != nil {
		return
	}
	if ic.config.MaxFileSize > 0 && size > ic.config.MaxFileSize {
		warnLog.Printf("File size %d exceeds max file size. file content omitted.", size)
		encoder.Close()
		return
	}
	if err = encoder.Close(); err != nil {
		return
	}
	op.Data["file"] = string(gridByteBuffer.Bytes())
	return
}

func notMonstache(config *configOptions) gtm.OpFilter {
	db := config.ConfigDatabaseName
	return func(op *gtm.Op) bool {
		return op.GetDatabase() != db
	}
}

func notChunks(op *gtm.Op) bool {
	return !chunksRegex.MatchString(op.GetCollection())
}

func notConfig(op *gtm.Op) bool {
	return op.GetDatabase() != "config"
}

func notSystem(op *gtm.Op) bool {
	return !systemsRegex.MatchString(op.GetCollection())
}

func filterWithRegex(regex string) gtm.OpFilter {
	var validNameSpace = regexp.MustCompile(regex)
	return func(op *gtm.Op) bool {
		if op.IsDrop() {
			return true
		}
		return validNameSpace.MatchString(op.Namespace)
	}
}

func filterDropWithRegex(regex string) gtm.OpFilter {
	var validNameSpace = regexp.MustCompile(regex)
	return func(op *gtm.Op) bool {
		if op.IsDrop() {
			return validNameSpace.MatchString(op.Namespace)
		}
		return true
	}
}

func filterWithPlugin() gtm.OpFilter {
	return func(op *gtm.Op) bool {
		var keep = true
		if (op.IsInsert() || op.IsUpdate()) && op.Data != nil {
			keep = false
			input := &monstachemap.MapperPluginInput{
				Document:          op.Data,
				Namespace:         op.Namespace,
				Database:          op.GetDatabase(),
				Collection:        op.GetCollection(),
				Operation:         op.Operation,
				UpdateDescription: op.UpdateDescription,
			}
			if ok, err := filterPlugin(input); err == nil {
				keep = ok
			} else {
				errorLog.Println(err)
			}
		}
		return keep
	}
}

func filterWithScript() gtm.OpFilter {
	return func(op *gtm.Op) bool {
		var keep = true
		if (op.IsInsert() || op.IsUpdate()) && op.Data != nil {
			nss := []string{"", op.Namespace}
			for _, ns := range nss {
				if env := filterEnvs[ns]; env != nil {
					keep = false
					arg := convertMapJavascript(op.Data)
					arg2 := op.Namespace
					arg3 := convertMapJavascript(op.UpdateDescription)
					env.lock.Lock()
					defer env.lock.Unlock()
					val, err := env.VM.Call("module.exports", arg, arg, arg2, arg3)
					if err != nil {
						errorLog.Println(err)
					} else {
						if ok, err := val.ToBoolean(); err == nil {
							keep = ok
						} else {
							errorLog.Println(err)
						}
					}
				}
				if !keep {
					break
				}
			}
		}
		return keep
	}
}

func filterInverseWithRegex(regex string) gtm.OpFilter {
	var invalidNameSpace = regexp.MustCompile(regex)
	return func(op *gtm.Op) bool {
		if op.IsDrop() {
			return true
		}
		return !invalidNameSpace.MatchString(op.Namespace)
	}
}

func filterDropInverseWithRegex(regex string) gtm.OpFilter {
	var invalidNameSpace = regexp.MustCompile(regex)
	return func(op *gtm.Op) bool {
		if op.IsDrop() {
			return !invalidNameSpace.MatchString(op.Namespace)
		}
		return true
	}
}

func (ic *indexClient) ensureClusterTTL() error {
	io := options.Index()
	io.SetName("expireAt")
	io.SetBackground(true)
	io.SetExpireAfterSeconds(30)
	im := mongo.IndexModel{
		Keys:    bson.M{"expireAt": 1},
		Options: io,
	}
	col := ic.mongo.Database(ic.config.ConfigDatabaseName).Collection("cluster")
	iv := col.Indexes()
	_, err := iv.CreateOne(context.Background(), im)
	return err
}

func (ic *indexClient) enableProcess() (bool, error) {
	var err error
	var host string
	col := ic.mongo.Database(ic.config.ConfigDatabaseName).Collection("cluster")
	findOneOpts := options.FindOne().SetProjection(bson.M{"_id": 1})
	sr := col.FindOne(context.Background(), bson.M{"_id": ic.config.ResumeName}, findOneOpts)
	err = sr.Err()
	if err != mongo.ErrNoDocuments {
		// only attempt the insert if no documents match
		return false, err
	}
	doc := bson.M{}
	doc["_id"] = ic.config.ResumeName
	doc["pid"] = os.Getpid()
	if host, err = os.Hostname(); err == nil {
		doc["host"] = host
	} else {
		return false, err
	}
	doc["expireAt"] = time.Now().UTC()
	_, err = col.InsertOne(context.Background(), doc)
	if err == nil {
		// update using $currentDate
		_, err = ic.ensureEnabled()
		if err == nil {
			return true, nil
		}
	}
	if mongo.IsDuplicateKeyError(err) {
		return false, nil
	}
	return false, err
}

func (ic *indexClient) resetClusterState() error {
	col := ic.mongo.Database(ic.config.ConfigDatabaseName).Collection("cluster")
	_, err := col.DeleteOne(context.Background(), bson.M{"_id": ic.config.ResumeName})
	return err
}

func (ic *indexClient) ensureEnabled() (enabled bool, err error) {
	col := ic.mongo.Database(ic.config.ConfigDatabaseName).Collection("cluster")
	result := col.FindOne(context.Background(), bson.M{
		"_id": ic.config.ResumeName,
	})
	if err = result.Err(); err == nil {
		doc := make(map[string]interface{})
		if err = result.Decode(&doc); err == nil {
			if doc["pid"] != nil && doc["host"] != nil {
				var hostname string
				pid := doc["pid"].(int32)
				host := doc["host"].(string)
				if hostname, err = os.Hostname(); err == nil {
					enabled = (int(pid) == os.Getpid() && host == hostname)
					if enabled {
						_, err = col.UpdateOne(context.Background(), bson.M{
							"_id": ic.config.ResumeName,
						}, bson.M{
							"$currentDate": bson.M{"expireAt": true},
						})
					}
				}
			}
		}
	}
	if err == mongo.ErrNoDocuments {
		err = nil
	}
	return
}

func (ic *indexClient) pauseWork() {
	ic.gtmCtx.Pause()
}

func (ic *indexClient) resumeWork() {
	col := ic.mongo.Database(ic.config.ConfigDatabaseName).Collection("monstache")
	result := col.FindOne(context.Background(), bson.M{
		"_id": ic.config.ResumeName,
	})
	if err := result.Err(); err == nil {
		doc := make(map[string]interface{})
		if err = result.Decode(&doc); err == nil {
			if doc["ts"] != nil {
				ts := doc["ts"].(primitive.Timestamp)
				ic.gtmCtx.Since(ts)
			}
		}
	}
	ic.gtmCtx.Resume()
}

func (ic *indexClient) saveTokens() error {
	var err error
	if len(ic.tokens) == 0 {
		return err
	}
	col := ic.mongo.Database(ic.config.ConfigDatabaseName).Collection("tokens")
	bwo := options.BulkWrite().SetOrdered(false)
	var models []mongo.WriteModel
	for streamID, token := range ic.tokens {
		filter := bson.M{
			"resumeName": ic.config.ResumeName,
			"streamID":   streamID,
		}
		replacement := bson.M{
			"resumeName": ic.config.ResumeName,
			"streamID":   streamID,
			"token":      token,
		}
		model := mongo.NewReplaceOneModel()
		model.SetUpsert(true)
		model.SetFilter(filter)
		model.SetReplacement(replacement)
		models = append(models, model)
	}
	_, err = col.BulkWrite(context.Background(), models, bwo)
	if err == nil {
		ic.tokens = bson.M{}
	}
	return err
}

func (ic *indexClient) saveTimestamp() error {
	col := ic.mongo.Database(ic.config.ConfigDatabaseName).Collection("monstache")
	doc := map[string]interface{}{
		"ts": ic.lastTs,
	}
	opts := options.Update()
	opts.SetUpsert(true)
	_, err := col.UpdateOne(context.Background(), bson.M{
		"_id": ic.config.ResumeName,
	}, bson.M{
		"$set": doc,
	}, opts)
	return err
}

func (ic *indexClient) filterDirectReadNamespaces(wanted []string) (results []string, err error) {
	results = make([]string, 0)
	col := ic.mongo.Database(ic.config.ConfigDatabaseName).Collection("directreads")
	filter := bson.M{
		"_id": ic.config.ResumeName,
	}
	result := col.FindOne(context.Background(), filter)
	if err = result.Err(); err == nil {
		var doc struct {
			Ns []string `bson:"ns"`
		}
		if err = result.Decode(&doc); err == nil {
			var ns, skipped []string
			if len(doc.Ns) > 0 {
				ns = doc.Ns
			}
			for _, name := range wanted {
				markedDone := false
				for _, n := range ns {
					if name == n {
						markedDone = true
						break
					}
				}
				if !markedDone {
					results = append(results, name)
				} else {
					skipped = append(skipped, name)
				}
			}
			if len(skipped) > 0 {
				infoLog.Printf("Skipping direct reads for namespaces marked complete: %+q", skipped)
			}
		}
	} else if err == mongo.ErrNoDocuments {
		err = nil
		results = wanted
	}
	return
}

func (ic *indexClient) saveDirectReadNamespaces() (err error) {
	col := ic.mongo.Database(ic.config.ConfigDatabaseName).Collection("directreads")
	filter := bson.M{
		"_id": ic.config.ResumeName,
	}
	ts := time.Now().UTC()
	update := bson.M{
		"$set":         bson.M{"updated": ts},
		"$setOnInsert": bson.M{"created": ts},
		"$addToSet":    bson.M{"ns": bson.M{"$each": ic.config.DirectReadNs}},
	}
	opts := options.Update().SetUpsert(true)
	_, err = col.UpdateOne(context.Background(), filter, update, opts)
	return
}

func (config *configOptions) parseCommandLineFlags() *configOptions {
	flag.BoolVar(&config.Print, "print-config", false, "Print the configuration and then exit")
	flag.BoolVar(&config.EnableTemplate, "tpl", false, "True to interpret the config file as a template")
	flag.StringVar(&config.EnvDelimiter, "env-delimiter", ",", "A delimiter to use when splitting environment variable values")
	flag.StringVar(&config.MongoURL, "mongo-url", "", "MongoDB server or router server connection URL")
	flag.StringVar(&config.MongoConfigURL, "mongo-config-url", "", "MongoDB config server connection URL")
	flag.StringVar(&config.MongoOpLogDatabaseName, "mongo-oplog-database-name", "", "Override the database name which contains the mongodb oplog")
	flag.StringVar(&config.MongoOpLogCollectionName, "mongo-oplog-collection-name", "", "Override the collection name which contains the mongodb oplog")
	flag.StringVar(&config.GraylogAddr, "graylog-addr", "", "Send logs to a Graylog server at this address")
	flag.StringVar(&config.ElasticVersion, "elasticsearch-version", "", "Specify elasticsearch version directly instead of getting it from the server")
	flag.StringVar(&config.ElasticUser, "elasticsearch-user", "", "The elasticsearch user name for basic auth")
	flag.StringVar(&config.ElasticPassword, "elasticsearch-password", "", "The elasticsearch password for basic auth")
	flag.StringVar(&config.ElasticPemFile, "elasticsearch-pem-file", "", "Path to a PEM file for secure connections to elasticsearch")
	flag.BoolVar(&config.ElasticValidatePemFile, "elasticsearch-validate-pem-file", true, "Set to boolean false to not validate the Elasticsearch PEM file")
	flag.IntVar(&config.ElasticMaxConns, "elasticsearch-max-conns", 0, "Elasticsearch max connections")
	flag.IntVar(&config.PostProcessors, "post-processors", 0, "Number of post-processing go routines")
	flag.IntVar(&config.FileDownloaders, "file-downloaders", 0, "GridFs download go routines")
	flag.IntVar(&config.RelateThreads, "relate-threads", 0, "Number of threads dedicated to processing relationships")
	flag.IntVar(&config.RelateBuffer, "relate-buffer", 0, "Number of relates to queue before skipping and reporting an error")
	flag.BoolVar(&config.ElasticRetry, "elasticsearch-retry", false, "True to retry failed request to Elasticsearch")
	flag.IntVar(&config.ElasticMaxDocs, "elasticsearch-max-docs", 0, "Number of docs to hold before flushing to Elasticsearch")
	flag.IntVar(&config.ElasticMaxBytes, "elasticsearch-max-bytes", 0, "Number of bytes to hold before flushing to Elasticsearch")
	flag.IntVar(&config.ElasticMaxSeconds, "elasticsearch-max-seconds", 0, "Number of seconds before flushing to Elasticsearch")
	flag.IntVar(&config.ElasticClientTimeout, "elasticsearch-client-timeout", 0, "Number of seconds before a request to Elasticsearch is timed out")
	flag.Int64Var(&config.MaxFileSize, "max-file-size", 0, "GridFs file content exceeding this limit in bytes will not be indexed in Elasticsearch")
	flag.StringVar(&config.ConfigFile, "f", "", "Location of configuration file")
	flag.BoolVar(&config.DroppedDatabases, "dropped-databases", true, "True to delete indexes from dropped databases")
	flag.BoolVar(&config.DroppedCollections, "dropped-collections", true, "True to delete indexes from dropped collections")
	flag.BoolVar(&config.Version, "version", false, "True to print the version number")
	flag.BoolVar(&config.Gzip, "gzip", false, "True to enable gzip for requests to Elasticsearch")
	flag.BoolVar(&config.Verbose, "verbose", false, "True to output verbose messages")
	flag.BoolVar(&config.Pprof, "pprof", false, "True to enable pprof endpoints")
	flag.BoolVar(&config.EnableOplog, "enable-oplog", false, "True to enable direct tailing of the oplog")
	flag.BoolVar(&config.DisableChangeEvents, "disable-change-events", false, "True to disable listening for changes.  You must provide direct-reads in this case")
	flag.BoolVar(&config.EnableEasyJSON, "enable-easy-json", false, "True to enable easy-json serialization")
	flag.BoolVar(&config.Stats, "stats", false, "True to print out statistics")
	flag.BoolVar(&config.IndexStats, "index-stats", false, "True to index stats in elasticsearch")
	flag.StringVar(&config.StatsDuration, "stats-duration", "", "The duration after which stats are logged")
	flag.StringVar(&config.StatsIndexFormat, "stats-index-format", "", "time.Time supported format to use for the stats index names")
	flag.BoolVar(&config.Resume, "resume", false, "True to capture the last timestamp of this run and resume on a subsequent run")
	flag.Var(&config.ResumeStrategy, "resume-strategy", "Strategy to use for resuming. 0=timestamp,1=token")
	flag.Int64Var(&config.ResumeFromTimestamp, "resume-from-timestamp", 0, "Timestamp to resume syncing from")
	flag.BoolVar(&config.ResumeFromEarliestTimestamp, "resume-from-earliest-timestamp", false, "Automatically select an earliest timestamp to resume syncing from")
	flag.BoolVar(&config.ResumeWriteUnsafe, "resume-write-unsafe", false, "True to speedup writes of the last timestamp synched for resuming at the cost of error checking")
	flag.BoolVar(&config.Replay, "replay", false, "True to replay all events from the oplog and index them in elasticsearch")
	flag.BoolVar(&config.IndexFiles, "index-files", false, "True to index gridfs files into elasticsearch. Requires the elasticsearch mapper-attachments (deprecated) or ingest-attachment plugin")
	flag.BoolVar(&config.DisableFilePipelinePut, "disable-file-pipeline-put", false, "True to disable auto-creation of the ingest plugin pipeline")
	flag.BoolVar(&config.IndexAsUpdate, "index-as-update", false, "True to index documents as updates instead of overwrites")
	flag.BoolVar(&config.FileHighlighting, "file-highlighting", false, "True to enable the ability to highlight search times for a file query")
	flag.BoolVar(&config.EnablePatches, "enable-patches", false, "True to include an json-patch field on updates")
	flag.BoolVar(&config.FailFast, "fail-fast", false, "True to exit if a single _bulk request fails")
	flag.BoolVar(&config.IndexOplogTime, "index-oplog-time", false, "True to add date/time information from the oplog to each document when indexing")
	flag.BoolVar(&config.ExitAfterDirectReads, "exit-after-direct-reads", false, "True to exit the program after reading directly from the configured namespaces")
	flag.StringVar(&config.MergePatchAttr, "merge-patch-attribute", "", "Attribute to store json-patch values under")
	flag.StringVar(&config.ResumeName, "resume-name", "", "Name under which to load/store the resume state. Defaults to 'default'")
	flag.StringVar(&config.ClusterName, "cluster-name", "", "Name of the monstache process cluster")
	flag.StringVar(&config.Worker, "worker", "", "The name of this worker in a multi-worker configuration")
	flag.StringVar(&config.MapperPluginPath, "mapper-plugin-path", "", "The path to a .so file to load as a document mapper plugin")
	flag.StringVar(&config.DirectReadExcludeRegex, "direct-read-dynamic-exclude-regex", "", "A regex to use for excluding namespaces when using dynamic direct reads")
	flag.StringVar(&config.DirectReadIncludeRegex, "direct-read-dynamic-include-regex", "", "A regex to use for including namespaces when using dynamic direct reads")
	flag.StringVar(&config.NsRegex, "namespace-regex", "", "A regex which is matched against an operation's namespace (<database>.<collection>).  Only operations which match are synched to elasticsearch")
	flag.StringVar(&config.NsDropRegex, "namespace-drop-regex", "", "A regex which is matched against a drop operation's namespace (<database>.<collection>).  Only drop operations which match are synched to elasticsearch")
	flag.StringVar(&config.NsExcludeRegex, "namespace-exclude-regex", "", "A regex which is matched against an operation's namespace (<database>.<collection>).  Only operations which do not match are synched to elasticsearch")
	flag.StringVar(&config.NsDropExcludeRegex, "namespace-drop-exclude-regex", "", "A regex which is matched against a drop operation's namespace (<database>.<collection>).  Only drop operations which do not match are synched to elasticsearch")
	flag.Var(&config.ChangeStreamNs, "change-stream-namespace", "A list of change stream namespaces")
	flag.Var(&config.DirectReadNs, "direct-read-namespace", "A list of direct read namespaces")
	flag.IntVar(&config.DirectReadSplitMax, "direct-read-split-max", 0, "Max number of times to split a collection for direct reads")
	flag.IntVar(&config.DirectReadConcur, "direct-read-concur", 0, "Max number of direct-read-namespaces to read concurrently. By default all givne are read concurrently")
	flag.BoolVar(&config.DirectReadNoTimeout, "direct-read-no-timeout", false, "True to set the no cursor timeout flag for direct reads")
	flag.BoolVar(&config.DirectReadBounded, "direct-read-bounded", false, "True to limit direct reads to the docs present at query start time")
	flag.BoolVar(&config.DirectReadStateful, "direct-read-stateful", false, "True to mark direct read namespaces as complete and not sync them in future runs")
	flag.Var(&config.RoutingNamespaces, "routing-namespace", "A list of namespaces that override routing information")
	flag.Var(&config.TimeMachineNamespaces, "time-machine-namespace", "A list of direct read namespaces")
	flag.StringVar(&config.TimeMachineIndexPrefix, "time-machine-index-prefix", "", "A prefix to preprend to time machine indexes")
	flag.StringVar(&config.TimeMachineIndexSuffix, "time-machine-index-suffix", "", "A suffix to append to time machine indexes")
	flag.BoolVar(&config.TimeMachineDirectReads, "time-machine-direct-reads", false, "True to index the results of direct reads into the any time machine indexes")
	flag.BoolVar(&config.PipeAllowDisk, "pipe-allow-disk", false, "True to allow MongoDB to use the disk for pipeline options with lots of results")
	flag.Var(&config.ElasticUrls, "elasticsearch-url", "A list of Elasticsearch URLs")
	flag.Var(&config.FileNamespaces, "file-namespace", "A list of file namespaces")
	flag.Var(&config.PatchNamespaces, "patch-namespace", "A list of patch namespaces")
	flag.Var(&config.Workers, "workers", "A list of worker names")
	flag.BoolVar(&config.EnableHTTPServer, "enable-http-server", false, "True to enable an internal http server")
	flag.StringVar(&config.HTTPServerAddr, "http-server-addr", "", "The address the internal http server listens on")
	flag.BoolVar(&config.PruneInvalidJSON, "prune-invalid-json", false, "True to omit values which do not serialize to JSON such as +Inf and -Inf and thus cause errors")
	flag.Var(&config.DeleteStrategy, "delete-strategy", "Stategy to use for deletes. 0=stateless,1=stateful,2=ignore")
	flag.StringVar(&config.DeleteIndexPattern, "delete-index-pattern", "", "An Elasticsearch index-pattern to restric the scope of stateless deletes")
	flag.StringVar(&config.ConfigDatabaseName, "config-database-name", "", "The MongoDB database name that monstache uses to store metadata")
	flag.StringVar(&config.OplogTsFieldName, "oplog-ts-field-name", "", "Field name to use for the oplog timestamp")
	flag.StringVar(&config.OplogDateFieldName, "oplog-date-field-name", "", "Field name to use for the oplog date")
	flag.StringVar(&config.OplogDateFieldFormat, "oplog-date-field-format", "", "Format to use for the oplog date")
	flag.BoolVar(&config.Debug, "debug", false, "True to enable verbose debug information")
	flag.Parse()
	return config
}

func (config *configOptions) loadReplacements() {
	if config.Relate != nil {
		for _, r := range config.Relate {
			if r.Namespace != "" || r.WithNamespace != "" {
				dbCol := strings.SplitN(r.WithNamespace, ".", 2)
				if len(dbCol) != 2 {
					errorLog.Fatalf("Replacement namespace is invalid: %s", r.WithNamespace)
				}
				database, collection := dbCol[0], dbCol[1]
				r := &relation{
					Namespace:      r.Namespace,
					WithNamespace:  r.WithNamespace,
					SrcField:       r.SrcField,
					MatchField:     r.MatchField,
					KeepSrc:        r.KeepSrc,
					DotNotation:    r.DotNotation,
					MaxDepth:       r.MaxDepth,
					MatchFieldType: r.MatchFieldType,
					db:             database,
					col:            collection,
				}
				if r.SrcField == "" {
					r.SrcField = "_id"
				}
				if r.MatchField == "" {
					r.MatchField = "_id"
				}
				relates[r.Namespace] = append(relates[r.Namespace], r)
			} else {
				errorLog.Fatalln("Relates must specify namespace and with-namespace")
			}
		}
	}
}

func (config *configOptions) loadIndexTypes() {
	if config.Mapping != nil {
		for _, m := range config.Mapping {
			if m.Namespace != "" && m.Index != "" {
				mapIndexTypes[m.Namespace] = &indexMapping{
					Namespace: m.Namespace,
					Index:     strings.ToLower(m.Index),
				}
			} else {
				errorLog.Fatalln("Mappings must specify namespace and index")
			}
		}
	}
}

func (config *configOptions) loadPipelines() {
	for _, s := range config.Pipeline {
		if s.Path == "" && s.Script == "" {
			errorLog.Fatalln("Pipelines must specify path or script attributes")
		}
		if s.Path != "" && s.Script != "" {
			errorLog.Fatalln("Pipelines must specify path or script but not both")
		}
		if s.Path != "" {
			if script, err := ioutil.ReadFile(s.Path); err == nil {
				s.Script = string(script[:])
			} else {
				errorLog.Fatalf("Unable to load pipeline at path %s: %s", s.Path, err)
			}
		}
		if _, exists := filterEnvs[s.Namespace]; exists {
			errorLog.Fatalf("Multiple pipelines with namespace: %s", s.Namespace)
		}
		env := &executionEnv{
			VM:     otto.New(),
			Script: s.Script,
			lock:   &sync.Mutex{},
		}
		if err := env.VM.Set("module", make(map[string]interface{})); err != nil {
			errorLog.Fatalln(err)
		}
		if _, err := env.VM.Run(env.Script); err != nil {
			errorLog.Fatalln(err)
		}
		val, err := env.VM.Run("module.exports")
		if err != nil {
			errorLog.Fatalln(err)
		} else if !val.IsFunction() {
			errorLog.Fatalln("module.exports must be a function")
		}
		pipeEnvs[s.Namespace] = env
	}
}

func (config *configOptions) loadFilters() {
	for _, s := range config.Filter {
		if s.Script != "" || s.Path != "" {
			if s.Path != "" && s.Script != "" {
				errorLog.Fatalln("Filters must specify path or script but not both")
			}
			if s.Path != "" {
				if script, err := ioutil.ReadFile(s.Path); err == nil {
					s.Script = string(script[:])
				} else {
					errorLog.Fatalf("Unable to load filter at path %s: %s", s.Path, err)
				}
			}
			if _, exists := filterEnvs[s.Namespace]; exists {
				errorLog.Fatalf("Multiple filters with namespace: %s", s.Namespace)
			}
			env := &executionEnv{
				VM:     otto.New(),
				Script: s.Script,
				lock:   &sync.Mutex{},
			}
			if err := env.VM.Set("module", make(map[string]interface{})); err != nil {
				errorLog.Fatalln(err)
			}
			if _, err := env.VM.Run(env.Script); err != nil {
				errorLog.Fatalln(err)
			}
			val, err := env.VM.Run("module.exports")
			if err != nil {
				errorLog.Fatalln(err)
			} else if !val.IsFunction() {
				errorLog.Fatalln("module.exports must be a function")
			}
			filterEnvs[s.Namespace] = env
		} else {
			errorLog.Fatalln("Filters must specify path or script attributes")
		}
	}
}

func (config *configOptions) loadScripts() {
	for _, s := range config.Script {
		if s.Script != "" || s.Path != "" {
			if s.Path != "" && s.Script != "" {
				errorLog.Fatalln("Scripts must specify path or script but not both")
			}
			if s.Path != "" {
				if script, err := ioutil.ReadFile(s.Path); err == nil {
					s.Script = string(script[:])
				} else {
					errorLog.Fatalf("Unable to load script at path %s: %s", s.Path, err)
				}
			}
			if _, exists := mapEnvs[s.Namespace]; exists {
				errorLog.Fatalf("Multiple scripts with namespace: %s", s.Namespace)
			}
			env := &executionEnv{
				VM:     otto.New(),
				Script: s.Script,
				lock:   &sync.Mutex{},
			}
			if err := env.VM.Set("module", make(map[string]interface{})); err != nil {
				errorLog.Fatalln(err)
			}
			if _, err := env.VM.Run(env.Script); err != nil {
				errorLog.Fatalln(err)
			}
			val, err := env.VM.Run("module.exports")
			if err != nil {
				errorLog.Fatalln(err)
			} else if !val.IsFunction() {
				errorLog.Fatalln("module.exports must be a function")
			}

			mapEnvs[s.Namespace] = env
			if s.Routing {
				routingNamespaces[s.Namespace] = true
			}
		} else {
			errorLog.Fatalln("Scripts must specify path or script")
		}
	}
}

func (config *configOptions) loadPlugins() *configOptions {
	if config.MapperPluginPath != "" {
		funcDefined := false
		p, err := plugin.Open(config.MapperPluginPath)
		if err != nil {
			errorLog.Fatalf("Unable to load mapper plugin %s: %s", config.MapperPluginPath, err)
		}
		initiator, err := p.Lookup("Init")
		if err == nil {
			funcDefined = true
			switch initiator.(type) {
			case func(*monstachemap.InitPluginInput) error:
				initPlugin = initiator.(func(*monstachemap.InitPluginInput) error)
			default:
				errorLog.Fatalf("Plugin 'Init' function must be typed %T", initPlugin)
			}
		}
		mapper, err := p.Lookup("Map")
		if err == nil {
			funcDefined = true
			switch mapper.(type) {
			case func(*monstachemap.MapperPluginInput) (*monstachemap.MapperPluginOutput, error):
				mapperPlugin = mapper.(func(*monstachemap.MapperPluginInput) (*monstachemap.MapperPluginOutput, error))
			default:
				errorLog.Fatalf("Plugin 'Map' function must be typed %T", mapperPlugin)
			}
		}
		filter, err := p.Lookup("Filter")
		if err == nil {
			funcDefined = true
			switch filter.(type) {
			case func(*monstachemap.MapperPluginInput) (bool, error):
				filterPlugin = filter.(func(*monstachemap.MapperPluginInput) (bool, error))
			default:
				errorLog.Fatalf("Plugin 'Filter' function must be typed %T", filterPlugin)
			}

		}
		process, err := p.Lookup("Process")
		if err == nil {
			funcDefined = true
			switch process.(type) {
			case func(*monstachemap.ProcessPluginInput) error:
				processPlugin = process.(func(*monstachemap.ProcessPluginInput) error)
			default:
				errorLog.Fatalf("Plugin 'Process' function must be typed %T", processPlugin)
			}
		}
		pipe, err := p.Lookup("Pipeline")
		if err == nil {
			funcDefined = true
			switch pipe.(type) {
			case func(string, bool) ([]interface{}, error):
				pipePlugin = pipe.(func(string, bool) ([]interface{}, error))
			default:
				errorLog.Fatalf("Plugin 'Pipeline' function must be typed %T", pipePlugin)
			}
		}
		if !funcDefined {
			warnLog.Println("Plugin loaded but did not find a Map, Filter, Process or Pipeline function")
		}
	}
	return config
}

func (config *configOptions) decodeAsTemplate() *configOptions {
	env := map[string]string{}
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) < 2 {
			continue
		}
		name, val := pair[0], pair[1]
		env[name] = val
	}
	tpl, err := ioutil.ReadFile(config.ConfigFile)
	if err != nil {
		errorLog.Fatalln(err)
	}
	var t = template.Must(template.New("config").Parse(string(tpl)))
	var b bytes.Buffer
	err = t.Execute(&b, env)
	if err != nil {
		errorLog.Fatalln(err)
	}
	if md, err := toml.Decode(b.String(), config); err != nil {
		errorLog.Fatalln(err)
	} else if ud := md.Undecoded(); len(ud) != 0 {
		errorLog.Fatalf("Config file contains undecoded keys: %q", ud)
	}
	return config
}

func (config *configOptions) loadConfigFile() *configOptions {
	if config.ConfigFile != "" {
		var tomlConfig = configOptions{
			ConfigFile:             config.ConfigFile,
			LogRotate:              config.LogRotate,
			DroppedDatabases:       true,
			DroppedCollections:     true,
			ElasticValidatePemFile: true,
			GtmSettings:            gtmDefaultSettings(),
		}
		if config.EnableTemplate {
			tomlConfig.decodeAsTemplate()
		} else {
			if md, err := toml.DecodeFile(tomlConfig.ConfigFile, &tomlConfig); err != nil {
				errorLog.Fatalln(err)
			} else if ud := md.Undecoded(); len(ud) != 0 {
				errorLog.Fatalf("Config file contains undecoded keys: %q", ud)
			}
		}
		if config.MongoURL == "" {
			config.MongoURL = tomlConfig.MongoURL
		}
		if config.MongoConfigURL == "" {
			config.MongoConfigURL = tomlConfig.MongoConfigURL
		}
		if config.MongoOpLogDatabaseName == "" {
			config.MongoOpLogDatabaseName = tomlConfig.MongoOpLogDatabaseName
		}
		if config.MongoOpLogCollectionName == "" {
			config.MongoOpLogCollectionName = tomlConfig.MongoOpLogCollectionName
		}
		if config.ElasticUser == "" {
			config.ElasticUser = tomlConfig.ElasticUser
		}
		if config.ElasticPassword == "" {
			config.ElasticPassword = tomlConfig.ElasticPassword
		}
		if config.ElasticPemFile == "" {
			config.ElasticPemFile = tomlConfig.ElasticPemFile
		}
		if config.ElasticValidatePemFile && !tomlConfig.ElasticValidatePemFile {
			config.ElasticValidatePemFile = false
		}
		if config.ElasticVersion == "" {
			config.ElasticVersion = tomlConfig.ElasticVersion
		}
		if config.ElasticMaxConns == 0 {
			config.ElasticMaxConns = tomlConfig.ElasticMaxConns
		}
		if config.ElasticHealth0 == 0 {
			config.ElasticHealth0 = tomlConfig.ElasticHealth0
		}
		if config.ElasticHealth1 == 0 {
			config.ElasticHealth1 = tomlConfig.ElasticHealth1
		}
		if config.DirectReadSplitMax == 0 {
			config.DirectReadSplitMax = tomlConfig.DirectReadSplitMax
		}
		if config.DirectReadConcur == 0 {
			config.DirectReadConcur = tomlConfig.DirectReadConcur
		}
		if !config.DirectReadNoTimeout && tomlConfig.DirectReadNoTimeout {
			config.DirectReadNoTimeout = true
		}
		if !config.DirectReadBounded && tomlConfig.DirectReadBounded {
			config.DirectReadBounded = true
		}
		if !config.DirectReadStateful && tomlConfig.DirectReadStateful {
			config.DirectReadStateful = true
		}
		if !config.ElasticRetry && tomlConfig.ElasticRetry {
			config.ElasticRetry = true
		}
		if config.ElasticMaxDocs == 0 {
			config.ElasticMaxDocs = tomlConfig.ElasticMaxDocs
		}
		if config.ElasticMaxBytes == 0 {
			config.ElasticMaxBytes = tomlConfig.ElasticMaxBytes
		}
		if config.ElasticMaxSeconds == 0 {
			config.ElasticMaxSeconds = tomlConfig.ElasticMaxSeconds
		}
		if config.ElasticClientTimeout == 0 {
			config.ElasticClientTimeout = tomlConfig.ElasticClientTimeout
		}
		if config.MaxFileSize == 0 {
			config.MaxFileSize = tomlConfig.MaxFileSize
		}
		if !config.IndexFiles {
			config.IndexFiles = tomlConfig.IndexFiles
		}
		if !config.DisableFilePipelinePut {
			config.DisableFilePipelinePut = tomlConfig.DisableFilePipelinePut
		}
		if config.FileDownloaders == 0 {
			config.FileDownloaders = tomlConfig.FileDownloaders
		}
		if config.RelateThreads == 0 {
			config.RelateThreads = tomlConfig.RelateThreads
		}
		if config.RelateBuffer == 0 {
			config.RelateBuffer = tomlConfig.RelateBuffer
		}
		if config.PostProcessors == 0 {
			config.PostProcessors = tomlConfig.PostProcessors
		}
		if config.DeleteStrategy == 0 {
			config.DeleteStrategy = tomlConfig.DeleteStrategy
		}
		if config.DeleteIndexPattern == "" {
			config.DeleteIndexPattern = tomlConfig.DeleteIndexPattern
		}
		if config.DroppedDatabases && !tomlConfig.DroppedDatabases {
			config.DroppedDatabases = false
		}
		if config.DroppedCollections && !tomlConfig.DroppedCollections {
			config.DroppedCollections = false
		}
		if !config.Gzip && tomlConfig.Gzip {
			config.Gzip = true
		}
		if !config.Verbose && tomlConfig.Verbose {
			config.Verbose = true
		}
		if !config.Stats && tomlConfig.Stats {
			config.Stats = true
		}
		if !config.Pprof && tomlConfig.Pprof {
			config.Pprof = true
		}
		if !config.EnableOplog && tomlConfig.EnableOplog {
			config.EnableOplog = true
		}
		if !config.EnableEasyJSON && tomlConfig.EnableEasyJSON {
			config.EnableEasyJSON = true
		}
		if !config.DisableChangeEvents && tomlConfig.DisableChangeEvents {
			config.DisableChangeEvents = true
		}
		if !config.IndexStats && tomlConfig.IndexStats {
			config.IndexStats = true
		}
		if config.StatsDuration == "" {
			config.StatsDuration = tomlConfig.StatsDuration
		}
		if config.StatsIndexFormat == "" {
			config.StatsIndexFormat = tomlConfig.StatsIndexFormat
		}
		if !config.IndexAsUpdate && tomlConfig.IndexAsUpdate {
			config.IndexAsUpdate = true
		}
		if !config.FileHighlighting && tomlConfig.FileHighlighting {
			config.FileHighlighting = true
		}
		if !config.EnablePatches && tomlConfig.EnablePatches {
			config.EnablePatches = true
		}
		if !config.PruneInvalidJSON && tomlConfig.PruneInvalidJSON {
			config.PruneInvalidJSON = true
		}
		if !config.Debug && tomlConfig.Debug {
			config.Debug = true
		}
		if !config.Replay && tomlConfig.Replay {
			config.Replay = true
		}
		if !config.Resume && tomlConfig.Resume {
			config.Resume = true
		}
		if !config.ResumeWriteUnsafe && tomlConfig.ResumeWriteUnsafe {
			config.ResumeWriteUnsafe = true
		}
		if config.ResumeFromTimestamp == 0 {
			config.ResumeFromTimestamp = tomlConfig.ResumeFromTimestamp
		}
		if !config.ResumeFromEarliestTimestamp && tomlConfig.ResumeFromEarliestTimestamp {
			config.ResumeFromEarliestTimestamp = true
		}
		if config.MergePatchAttr == "" {
			config.MergePatchAttr = tomlConfig.MergePatchAttr
		}
		if !config.FailFast && tomlConfig.FailFast {
			config.FailFast = true
		}
		if !config.IndexOplogTime && tomlConfig.IndexOplogTime {
			config.IndexOplogTime = true
		}
		if config.OplogTsFieldName == "" {
			config.OplogTsFieldName = tomlConfig.OplogTsFieldName
		}
		if config.OplogDateFieldName == "" {
			config.OplogDateFieldName = tomlConfig.OplogDateFieldName
		}
		if config.OplogDateFieldFormat == "" {
			config.OplogDateFieldFormat = tomlConfig.OplogDateFieldFormat
		}
		if config.ConfigDatabaseName == "" {
			config.ConfigDatabaseName = tomlConfig.ConfigDatabaseName
		}
		if !config.ExitAfterDirectReads && tomlConfig.ExitAfterDirectReads {
			config.ExitAfterDirectReads = true
		}
		if config.ResumeName == "" {
			config.ResumeName = tomlConfig.ResumeName
		}
		if config.ClusterName == "" {
			config.ClusterName = tomlConfig.ClusterName
		}
		if config.ResumeStrategy == 0 {
			config.ResumeStrategy = tomlConfig.ResumeStrategy
		}
		if config.DirectReadExcludeRegex == "" {
			config.DirectReadExcludeRegex = tomlConfig.DirectReadExcludeRegex
		}
		if config.DirectReadIncludeRegex == "" {
			config.DirectReadIncludeRegex = tomlConfig.DirectReadIncludeRegex
		}
		if config.NsRegex == "" {
			config.NsRegex = tomlConfig.NsRegex
		}
		if config.NsDropRegex == "" {
			config.NsDropRegex = tomlConfig.NsDropRegex
		}
		if config.NsExcludeRegex == "" {
			config.NsExcludeRegex = tomlConfig.NsExcludeRegex
		}
		if config.NsDropExcludeRegex == "" {
			config.NsDropExcludeRegex = tomlConfig.NsDropExcludeRegex
		}
		if config.IndexFiles {
			if len(config.FileNamespaces) == 0 {
				config.FileNamespaces = tomlConfig.FileNamespaces
				config.loadGridFsConfig()
			}
		}
		if config.Worker == "" {
			config.Worker = tomlConfig.Worker
		}
		if config.GraylogAddr == "" {
			config.GraylogAddr = tomlConfig.GraylogAddr
		}
		if config.MapperPluginPath == "" {
			config.MapperPluginPath = tomlConfig.MapperPluginPath
		}
		if config.EnablePatches {
			if len(config.PatchNamespaces) == 0 {
				config.PatchNamespaces = tomlConfig.PatchNamespaces
				config.loadPatchNamespaces()
			}
		}
		if len(config.RoutingNamespaces) == 0 {
			config.RoutingNamespaces = tomlConfig.RoutingNamespaces
			config.loadRoutingNamespaces()
		}
		if len(config.TimeMachineNamespaces) == 0 {
			config.TimeMachineNamespaces = tomlConfig.TimeMachineNamespaces
			config.loadTimeMachineNamespaces()
		}
		if config.TimeMachineIndexPrefix == "" {
			config.TimeMachineIndexPrefix = tomlConfig.TimeMachineIndexPrefix
		}
		if config.TimeMachineIndexSuffix == "" {
			config.TimeMachineIndexSuffix = tomlConfig.TimeMachineIndexSuffix
		}
		if !config.TimeMachineDirectReads {
			config.TimeMachineDirectReads = tomlConfig.TimeMachineDirectReads
		}
		if !config.PipeAllowDisk {
			config.PipeAllowDisk = tomlConfig.PipeAllowDisk
		}
		if len(config.DirectReadNs) == 0 {
			config.DirectReadNs = tomlConfig.DirectReadNs
		}
		if len(config.ChangeStreamNs) == 0 {
			config.ChangeStreamNs = tomlConfig.ChangeStreamNs
		}
		if len(config.ElasticUrls) == 0 {
			config.ElasticUrls = tomlConfig.ElasticUrls
		}
		if len(config.Workers) == 0 {
			config.Workers = tomlConfig.Workers
		}
		if !config.EnableHTTPServer && tomlConfig.EnableHTTPServer {
			config.EnableHTTPServer = true
		}
		if config.HTTPServerAddr == "" {
			config.HTTPServerAddr = tomlConfig.HTTPServerAddr
		}
		if !config.AWSConnect.enabled() {
			config.AWSConnect = tomlConfig.AWSConnect
		}
		if !config.Logs.enabled() {
			config.Logs = tomlConfig.Logs
		}
		if !config.ElasticPKIAuth.enabled() {
			config.ElasticPKIAuth = tomlConfig.ElasticPKIAuth
		}
		config.GtmSettings = tomlConfig.GtmSettings
		config.Relate = tomlConfig.Relate
		config.LogRotate = tomlConfig.LogRotate
		tomlConfig.loadScripts()
		tomlConfig.loadFilters()
		tomlConfig.loadPipelines()
		tomlConfig.loadIndexTypes()
		tomlConfig.loadReplacements()
	}
	return config
}

func (config *configOptions) newLogger(path string) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   path,
		MaxSize:    config.LogRotate.MaxSize,
		MaxBackups: config.LogRotate.MaxBackups,
		MaxAge:     config.LogRotate.MaxAge,
		LocalTime:  config.LogRotate.LocalTime,
		Compress:   config.LogRotate.Compress,
	}
}

func (config *configOptions) setupLogging() *configOptions {
	if config.GraylogAddr != "" {
		gelfWriter, err := gelf.NewUDPWriter(config.GraylogAddr)
		if err != nil {
			errorLog.Fatalf("Error creating gelf writer: %s", err)
		}
		infoLog.SetOutput(gelfWriter)
		warnLog.SetOutput(gelfWriter)
		errorLog.SetOutput(gelfWriter)
		traceLog.SetOutput(gelfWriter)
		statsLog.SetOutput(gelfWriter)
	} else {
		logs := config.Logs
		if logs.Info != "" {
			infoLog.SetOutput(config.newLogger(logs.Info))
		}
		if logs.Warn != "" {
			warnLog.SetOutput(config.newLogger(logs.Warn))
		}
		if logs.Error != "" {
			errorLog.SetOutput(config.newLogger(logs.Error))
		}
		if logs.Trace != "" {
			traceLog.SetOutput(config.newLogger(logs.Trace))
		}
		if logs.Stats != "" {
			statsLog.SetOutput(config.newLogger(logs.Stats))
		}
	}
	return config
}

func (config *configOptions) build() *configOptions {
	config.loadEnvironment()
	config.loadTimeMachineNamespaces()
	config.loadRoutingNamespaces()
	config.loadPatchNamespaces()
	config.loadGridFsConfig()
	config.loadConfigFile()
	config.loadPlugins()
	config.setDefaults()
	return config
}

func (config *configOptions) loadEnvironment() *configOptions {
	del := config.EnvDelimiter
	if del == "" {
		del = ","
	}
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) < 2 {
			continue
		}
		name, val := pair[0], pair[1]
		if val == "" {
			continue
		}
		if strings.HasSuffix(name, "__FILE") {
			var err error
			name, val, err = config.loadVariableValueFromFile(name, val)
			if err != nil {
				panic(err)
			}
		}
		switch name {
		case "MONSTACHE_MONGO_URL":
			if config.MongoURL == "" {
				config.MongoURL = val
			}
			break
		case "MONSTACHE_MONGO_CONFIG_URL":
			if config.MongoConfigURL == "" {
				config.MongoConfigURL = val
			}
			break
		case "MONSTACHE_MONGO_OPLOG_DB":
			if config.MongoOpLogDatabaseName == "" {
				config.MongoOpLogDatabaseName = val
			}
			break
		case "MONSTACHE_MONGO_OPLOG_COL":
			if config.MongoOpLogCollectionName == "" {
				config.MongoOpLogCollectionName = val
			}
			break
		case "MONSTACHE_ES_URLS":
			if len(config.ElasticUrls) == 0 {
				config.ElasticUrls = strings.Split(val, del)
			}
			break
		case "MONSTACHE_ES_USER":
			if config.ElasticUser == "" {
				config.ElasticUser = val
			}
			break
		case "MONSTACHE_ES_PASS":
			if config.ElasticPassword == "" {
				config.ElasticPassword = val
			}
			break
		case "MONSTACHE_ES_PEM":
			if config.ElasticPemFile == "" {
				config.ElasticPemFile = val
			}
			break
		case "MONSTACHE_ES_PKI_CERT":
			if config.ElasticPKIAuth.CertFile == "" {
				config.ElasticPKIAuth.CertFile = val
			}
			break
		case "MONSTACHE_ES_PKI_KEY":
			if config.ElasticPKIAuth.KeyFile == "" {
				config.ElasticPKIAuth.KeyFile = val
			}
			break
		case "MONSTACHE_ES_VALIDATE_PEM":
			v, err := strconv.ParseBool(val)
			if err != nil {
				errorLog.Fatalf("Failed to load MONSTACHE_ES_VALIDATE_PEM: %s", err)
			}
			config.ElasticValidatePemFile = v
			break
		case "MONSTACHE_WORKER":
			if config.Worker == "" {
				config.Worker = val
			}
			break
		case "MONSTACHE_CLUSTER":
			if config.ClusterName == "" {
				config.ClusterName = val
			}
			break
		case "MONSTACHE_DIRECT_READ_NS":
			if len(config.DirectReadNs) == 0 {
				config.DirectReadNs = strings.Split(val, del)
			}
			break
		case "MONSTACHE_CHANGE_STREAM_NS":
			if len(config.ChangeStreamNs) == 0 {
				config.ChangeStreamNs = strings.Split(val, del)
			}
			break
		case "MONSTACHE_DIRECT_READ_NS_DYNAMIC_EXCLUDE_REGEX":
			if config.DirectReadExcludeRegex == "" {
				config.DirectReadExcludeRegex = val
			}
			break
		case "MONSTACHE_DIRECT_READ_NS_DYNAMIC_INCLUDE_REGEX":
			if config.DirectReadIncludeRegex == "" {
				config.DirectReadIncludeRegex = val
			}
			break
		case "MONSTACHE_NS_REGEX":
			if config.NsRegex == "" {
				config.NsRegex = val
			}
			break
		case "MONSTACHE_NS_EXCLUDE_REGEX":
			if config.NsExcludeRegex == "" {
				config.NsExcludeRegex = val
			}
			break
		case "MONSTACHE_NS_DROP_REGEX":
			if config.NsDropRegex == "" {
				config.NsDropRegex = val
			}
			break
		case "MONSTACHE_NS_DROP_EXCLUDE_REGEX":
			if config.NsDropExcludeRegex == "" {
				config.NsDropExcludeRegex = val
			}
			break
		case "MONSTACHE_GRAYLOG_ADDR":
			if config.GraylogAddr == "" {
				config.GraylogAddr = val
			}
			break
		case "MONSTACHE_AWS_ACCESS_KEY":
			config.AWSConnect.AccessKey = val
			break
		case "MONSTACHE_AWS_SECRET_KEY":
			config.AWSConnect.SecretKey = val
			break
		case "MONSTACHE_AWS_REGION":
			config.AWSConnect.Region = val
			break
		case "MONSTACHE_LOG_DIR":
			config.Logs.Info = val + "/info.log"
			config.Logs.Warn = val + "/warn.log"
			config.Logs.Error = val + "/error.log"
			config.Logs.Trace = val + "/trace.log"
			config.Logs.Stats = val + "/stats.log"
			break
		case "MONSTACHE_LOG_MAX_SIZE":
			i, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				errorLog.Fatalf("Failed to load MONSTACHE_LOG_MAX_SIZE: %s", err)
			}
			config.LogRotate.MaxSize = int(i)
			break
		case "MONSTACHE_LOG_MAX_BACKUPS":
			i, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				errorLog.Fatalf("Failed to load MONSTACHE_LOG_MAX_BACKUPS: %s", err)
			}
			config.LogRotate.MaxBackups = int(i)
			break
		case "MONSTACHE_LOG_MAX_AGE":
			i, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				errorLog.Fatalf("Failed to load MONSTACHE_LOG_MAX_AGE: %s", err)
			}
			config.LogRotate.MaxAge = int(i)
			break
		case "MONSTACHE_HTTP_ADDR":
			if config.HTTPServerAddr == "" {
				config.HTTPServerAddr = val
			}
			break
		case "MONSTACHE_FILE_NS":
			if len(config.FileNamespaces) == 0 {
				config.FileNamespaces = strings.Split(val, del)
			}
			break
		case "MONSTACHE_PATCH_NS":
			if len(config.PatchNamespaces) == 0 {
				config.PatchNamespaces = strings.Split(val, del)
			}
			break
		case "MONSTACHE_TIME_MACHINE_NS":
			if len(config.TimeMachineNamespaces) == 0 {
				config.TimeMachineNamespaces = strings.Split(val, del)
			}
			break
		default:
			continue
		}
	}
	return config
}

func (config *configOptions) loadVariableValueFromFile(name string, path string) (n string, v string, err error) {
	name = strings.TrimSuffix(name, "__FILE")
	f, err := os.Open(path)
	if err != nil {
		return name, "", fmt.Errorf("read value for %s from file failed: %s", name, err)
	}
	defer f.Close()
	c, err := ioutil.ReadAll(f)
	if err != nil {
		return name, "", fmt.Errorf("read value for %s from file failed: %s", name, err)
	}
	return name, string(c), nil
}

func (config *configOptions) loadRoutingNamespaces() *configOptions {
	for _, namespace := range config.RoutingNamespaces {
		routingNamespaces[namespace] = true
	}
	return config
}

func (config *configOptions) loadTimeMachineNamespaces() *configOptions {
	for _, namespace := range config.TimeMachineNamespaces {
		tmNamespaces[namespace] = true
	}
	return config
}

func (config *configOptions) loadPatchNamespaces() *configOptions {
	for _, namespace := range config.PatchNamespaces {
		patchNamespaces[namespace] = true
	}
	return config
}

func (config *configOptions) loadGridFsConfig() *configOptions {
	for _, namespace := range config.FileNamespaces {
		fileNamespaces[namespace] = true
	}
	return config
}

func (config configOptions) dump() {
	if config.MongoURL != "" {
		config.MongoURL = cleanMongoURL(config.MongoURL)
	}
	if config.MongoConfigURL != "" {
		config.MongoConfigURL = cleanMongoURL(config.MongoConfigURL)
	}
	if config.ElasticUser != "" {
		config.ElasticUser = redact
	}
	if config.ElasticPassword != "" {
		config.ElasticPassword = redact
	}
	if config.AWSConnect.AccessKey != "" {
		config.AWSConnect.AccessKey = redact
	}
	if config.AWSConnect.SecretKey != "" {
		config.AWSConnect.SecretKey = redact
	}
	if config.AWSConnect.Region != "" {
		config.AWSConnect.Region = redact
	}
	json, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		errorLog.Printf("Unable to print configuration: %s", err)
	} else {
		infoLog.Println(string(json))
	}
}

func (config *configOptions) validate() {
	if config.DisableChangeEvents && len(config.DirectReadNs) == 0 {
		errorLog.Fatalln("Direct read namespaces must be specified if change events are disabled")
	}
	if config.AWSConnect.enabled() {
		if err := config.AWSConnect.validate(); err != nil {
			errorLog.Fatalln(err)
		}
	}
	if len(config.DirectReadNs) > 0 {
		if config.ElasticMaxSeconds < 5 {
			warnLog.Println("Direct read performance degrades with small values for elasticsearch-max-seconds. Set to 5s or greater to remove this warning.")
		}
		if config.ElasticMaxDocs > 0 {
			warnLog.Println("For performance reasons it is recommended to use elasticsearch-max-bytes instead of elasticsearch-max-docs since doc size may vary")
		}
	}
	if config.StatsDuration != "" {
		_, err := time.ParseDuration(config.StatsDuration)
		if err != nil {
			errorLog.Fatalf("Unable to parse stats duration: %s", err)
		}
	}
}

func (config *configOptions) setDefaults() *configOptions {
	if !config.EnableOplog && len(config.ChangeStreamNs) == 0 {
		config.ChangeStreamNs = []string{""}
	}
	if config.DisableChangeEvents {
		config.ChangeStreamNs = []string{}
		config.EnableOplog = false
	}
	if config.MongoURL == "" {
		config.MongoURL = mongoURLDefault
	}
	if config.ClusterName != "" {
		if config.Worker != "" {
			config.ResumeName = fmt.Sprintf("%s:%s", config.ClusterName, config.Worker)
		} else {
			config.ResumeName = config.ClusterName
		}
		config.Resume = true
	} else if config.Worker != "" {
		config.ResumeName = config.Worker
	} else if config.ResumeName == "" {
		config.ResumeName = resumeNameDefault
	}
	if config.ElasticMaxConns == 0 {
		config.ElasticMaxConns = elasticMaxConnsDefault
	}
	if config.ElasticClientTimeout == 0 {
		config.ElasticClientTimeout = elasticClientTimeoutDefault
	}
	if config.MergePatchAttr == "" {
		config.MergePatchAttr = "json-merge-patches"
	}
	if config.ElasticMaxSeconds == 0 {
		if len(config.DirectReadNs) > 0 {
			config.ElasticMaxSeconds = 5
		} else {
			config.ElasticMaxSeconds = 1
		}
	}
	if config.ElasticMaxDocs == 0 {
		config.ElasticMaxDocs = elasticMaxDocsDefault
	}
	if config.ElasticMaxBytes == 0 {
		config.ElasticMaxBytes = elasticMaxBytesDefault
	}
	if config.ElasticHealth0 == 0 {
		config.ElasticHealth0 = 15
	}
	if config.ElasticHealth1 == 0 {
		config.ElasticHealth1 = 5
	}
	if config.HTTPServerAddr == "" {
		config.HTTPServerAddr = ":8080"
	}
	if config.StatsIndexFormat == "" {
		config.StatsIndexFormat = "monstache.stats.2006-01-02"
	}
	if config.TimeMachineIndexPrefix == "" {
		config.TimeMachineIndexPrefix = "log"
	}
	if config.TimeMachineIndexSuffix == "" {
		config.TimeMachineIndexSuffix = "2006-01-02"
	}
	if config.DeleteIndexPattern == "" {
		config.DeleteIndexPattern = "*"
	}
	if config.FileDownloaders == 0 && config.IndexFiles {
		config.FileDownloaders = fileDownloadersDefault
	}
	if config.RelateThreads == 0 {
		config.RelateThreads = relateThreadsDefault
	}
	if config.RelateBuffer == 0 {
		config.RelateBuffer = relateBufferDefault
	}
	if config.PostProcessors == 0 && processPlugin != nil {
		config.PostProcessors = postProcessorsDefault
	}
	if config.OplogTsFieldName == "" {
		config.OplogTsFieldName = "oplog_ts"
	}
	if config.OplogDateFieldName == "" {
		config.OplogDateFieldName = "oplog_date"
	}
	if config.OplogDateFieldFormat == "" {
		config.OplogDateFieldFormat = "2006/01/02 15:04:05"
	}
	if config.ConfigDatabaseName == "" {
		config.ConfigDatabaseName = configDatabaseNameDefault
	}
	if config.ResumeFromTimestamp > 0 {
		if config.ResumeFromTimestamp <= math.MaxInt32 {
			config.ResumeFromTimestamp = config.ResumeFromTimestamp << 32
		}
	}
	return config
}

func cleanMongoURL(URL string) string {
	const (
		scheme    = "mongodb://"
		schemeSrv = "mongodb+srv://"
	)
	url := URL
	hasScheme := strings.HasPrefix(url, scheme)
	hasSchemeSrv := strings.HasPrefix(url, schemeSrv)
	url = strings.TrimPrefix(url, scheme)
	url = strings.TrimPrefix(url, schemeSrv)
	userEnd := strings.IndexAny(url, "@")
	if userEnd != -1 {
		url = redact + "@" + url[userEnd+1:]
	}
	if hasScheme {
		url = scheme + url
	} else if hasSchemeSrv {
		url = schemeSrv + url
	}
	return url
}

func (config *configOptions) dialMongo(URL string) (*mongo.Client, error) {
	var clientOptions *options.ClientOptions
	if config.mongoClientOptions == nil {
		// use the initial URL to create most of the client options
		// save the client options for potential use later with shards
		rb := bson.NewRegistryBuilder()
		rb.RegisterTypeMapEntry(bsontype.DateTime, reflect.TypeOf(time.Time{}))
		reg := rb.Build()
		clientOptions = options.Client()
		clientOptions.ApplyURI(URL)
		clientOptions.SetAppName("monstache")
		clientOptions.SetRegistry(reg)
		config.mongoClientOptions = clientOptions
	} else {
		// subsequent client connections will only be for adding shards
		// for shards we only have the hostname and replica set
		// apply the hostname to the previously saved client options
		clientOptions = config.mongoClientOptions
		clientOptions.ApplyURI(URL)
	}
	client, err := mongo.NewClient(clientOptions)
	if err != nil {
		return nil, err
	}
	err = client.Connect(context.Background())
	if err != nil {
		return nil, err
	}
	err = client.Ping(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (config *configOptions) NewHTTPClient() (client *http.Client, err error) {
	tlsConfig := &tls.Config{}
	if config.ElasticPemFile != "" {
		var ca []byte
		certs := x509.NewCertPool()
		if ca, err = ioutil.ReadFile(config.ElasticPemFile); err == nil {
			if ok := certs.AppendCertsFromPEM(ca); !ok {
				errorLog.Printf("No certs parsed successfully from %s", config.ElasticPemFile)
			}
			tlsConfig.RootCAs = certs
		} else {
			return client, err
		}
	}
	clientAuth := config.ElasticPKIAuth
	if clientAuth.enabled() {
		if err = clientAuth.validate(); err != nil {
			return client, err
		}
		var clientCert tls.Certificate
		clientCert, err = tls.LoadX509KeyPair(clientAuth.CertFile, clientAuth.KeyFile)
		if err != nil {
			return client, err
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}
	if config.ElasticValidatePemFile == false {
		// Turn off validation
		tlsConfig.InsecureSkipVerify = true
	}
	transport := &http.Transport{
		DisableCompression:  !config.Gzip,
		TLSHandshakeTimeout: time.Duration(30) * time.Second,
		TLSClientConfig:     tlsConfig,
	}
	client = &http.Client{
		Timeout:   time.Duration(config.ElasticClientTimeout) * time.Second,
		Transport: transport,
	}
	if config.AWSConnect.enabled() {
		var creds *credentials.Credentials
		if config.AWSConnect.Strategy == awsCredentialStrategyStatic {
			creds = credentials.NewStaticCredentials(config.AWSConnect.AccessKey, config.AWSConnect.SecretKey, "")
		} else if config.AWSConnect.Strategy == awsCredentialStrategyFile {
			creds = credentials.NewCredentials(&credentials.SharedCredentialsProvider{
				Filename: config.AWSConnect.CredentialsFile,
				Profile:  config.AWSConnect.Profile,
			})
		} else if config.AWSConnect.Strategy == awsCredentialStrategyEnv {
			creds = credentials.NewCredentials(&credentials.EnvProvider{})
		} else if config.AWSConnect.Strategy == awsCredentialStrategyEndpoint {
			creds = credentials.NewCredentials(defaults.RemoteCredProvider(*defaults.Config(), defaults.Handlers()))
		} else if config.AWSConnect.Strategy == awsCredentialStrategyChained {
			creds = credentials.NewChainCredentials([]credentials.Provider{
				&credentials.EnvProvider{},
				&credentials.SharedCredentialsProvider{
					Filename: config.AWSConnect.CredentialsFile,
					Profile:  config.AWSConnect.Profile,
				},
				defaults.RemoteCredProvider(*defaults.Config(), defaults.Handlers()),
			})
		}
		config.AWSConnect.creds = creds
		client = aws.NewV4SigningClientWithHTTPClient(creds, config.AWSConnect.Region, client)
	}
	return client, err
}

func (ic *indexClient) doDrop(op *gtm.Op) (err error) {
	if db, drop := op.IsDropDatabase(); drop {
		if ic.config.DroppedDatabases {
			if err = ic.deleteIndexes(db); err == nil {
				if e := ic.dropDBMeta(db); e != nil {
					errorLog.Printf("Unable to delete metadata for db: %s", e)
				}
			}
		}
	} else if col, drop := op.IsDropCollection(); drop {
		if ic.config.DroppedCollections {
			if err = ic.deleteIndex(op.GetDatabase() + "." + col); err == nil {
				if e := ic.dropCollectionMeta(op.GetDatabase() + "." + col); e != nil {
					errorLog.Printf("Unable to delete metadata for collection: %s", e)
				}
			}
		}
	}
	return
}

func (ic *indexClient) hasFileContent(op *gtm.Op) (ingest bool) {
	if !ic.config.IndexFiles {
		return
	}
	return fileNamespaces[op.Namespace]
}

func (ic *indexClient) addPatch(op *gtm.Op, objectID string,
	indexType *indexMapping, meta *indexingMeta) (err error) {
	var merges []interface{}
	var toJSON []byte
	if op.IsSourceDirect() {
		return nil
	}
	if op.Timestamp.T == 0 {
		return nil
	}
	client, config := ic.client, ic.config
	if op.IsUpdate() {
		ctx := context.Background()
		service := client.Get()
		service.Id(objectID)
		service.Index(indexType.Index)
		if meta.ID != "" {
			service.Id(meta.ID)
		}
		if meta.Index != "" {
			service.Index(meta.Index)
		}
		if meta.Routing != "" {
			service.Routing(meta.Routing)
		}
		if meta.Parent != "" {
			service.Parent(meta.Parent)
		}
		var resp *elastic.GetResult
		if resp, err = service.Do(ctx); err == nil {
			if resp.Found {
				var src map[string]interface{}
				if err = json.Unmarshal(resp.Source, &src); err == nil {
					if val, ok := src[config.MergePatchAttr]; ok {
						merges = val.([]interface{})
						for _, m := range merges {
							entry := m.(map[string]interface{})
							entry["ts"] = int(entry["ts"].(float64))
							entry["v"] = int(entry["v"].(float64))
						}
					}
					delete(src, config.MergePatchAttr)
					var fromJSON, mergeDoc []byte
					if fromJSON, err = json.Marshal(src); err == nil {
						if toJSON, err = json.Marshal(op.Data); err == nil {
							if mergeDoc, err = jsonpatch.CreateMergePatch(fromJSON, toJSON); err == nil {
								merge := make(map[string]interface{})
								merge["ts"] = op.Timestamp.T
								merge["p"] = string(mergeDoc)
								merge["v"] = len(merges) + 1
								merges = append(merges, merge)
								op.Data[config.MergePatchAttr] = merges
							}
						}
					}
				}
			} else {
				err = errors.New("Last document revision not found")
			}

		}
	} else {
		if _, found := op.Data[config.MergePatchAttr]; !found {
			if toJSON, err = json.Marshal(op.Data); err == nil {
				merge := make(map[string]interface{})
				merge["v"] = 1
				merge["ts"] = op.Timestamp.T
				merge["p"] = string(toJSON)
				merges = append(merges, merge)
				op.Data[config.MergePatchAttr] = merges
			}
		}
	}
	return
}

func (ic *indexClient) doIndexing(op *gtm.Op) (err error) {
	meta := parseIndexMeta(op)
	if meta.Skip {
		return
	}
	ic.prepareDataForIndexing(op)
	objectID, indexType := opIDToString(op), ic.mapIndex(op)
	if objectID == "" {
		return errors.New("Unable to index document due to empty _id value")
	}
	if ic.config.EnablePatches {
		if patchNamespaces[op.Namespace] {
			if e := ic.addPatch(op, objectID, indexType, meta); e != nil {
				errorLog.Printf("Unable to save json-patch info: %s", e)
			}
		}
	}
	ingestAttachment := false
	if ic.hasFileContent(op) {
		ingestAttachment = op.Data["file"] != nil
	}
	if ic.config.IndexAsUpdate && meta.Pipeline == "" && ingestAttachment == false {
		req := elastic.NewBulkUpdateRequest()
		req.UseEasyJSON(ic.config.EnableEasyJSON)
		req.Id(objectID)
		req.Index(indexType.Index)
		req.Doc(op.Data)
		req.DocAsUpsert(true)
		if meta.ID != "" {
			req.Id(meta.ID)
		}
		if meta.Index != "" {
			req.Index(meta.Index)
		}
		if meta.Type != "" {
		}
		if meta.Routing != "" {
			req.Routing(meta.Routing)
		}
		if meta.Parent != "" {
			req.Parent(meta.Parent)
		}
		if meta.RetryOnConflict != 0 {
			req.RetryOnConflict(meta.RetryOnConflict)
		}
		if _, err = req.Source(); err == nil {
			ic.bulk.Add(req)
		}
	} else {
		req := elastic.NewBulkIndexRequest()
		req.UseEasyJSON(ic.config.EnableEasyJSON)
		req.Id(objectID)
		req.Index(indexType.Index)
		req.Pipeline(indexType.Pipeline)
		req.Doc(op.Data)
		if meta.ID != "" {
			req.Id(meta.ID)
		}
		if meta.Index != "" {
			req.Index(meta.Index)
		}
		if meta.Routing != "" {
			req.Routing(meta.Routing)
		}
		if meta.Parent != "" {
			req.Parent(meta.Parent)
		}
		if meta.Version != 0 {
			req.Version(meta.Version)
		}
		if meta.VersionType != "" {
			req.VersionType(meta.VersionType)
		}
		if meta.Pipeline != "" {
			req.Pipeline(meta.Pipeline)
		}
		if meta.RetryOnConflict != 0 {
			req.RetryOnConflict(meta.RetryOnConflict)
		}
		if ingestAttachment {
			req.Pipeline("attachment")
		}
		if _, err = req.Source(); err == nil {
			ic.bulk.Add(req)
		}
	}

	if meta.shouldSave(ic.config) {
		if e := ic.setIndexMeta(op.Namespace, objectID, meta); e != nil {
			errorLog.Printf("Unable to save routing info: %s", e)
		}
	}

	if tmNamespaces[op.Namespace] {
		if op.IsSourceOplog() || ic.config.TimeMachineDirectReads {
			t := time.Now().UTC()
			tmIndex := func(idx string) string {
				pre, suf := ic.config.TimeMachineIndexPrefix, ic.config.TimeMachineIndexSuffix
				tmFormat := strings.Join([]string{pre, idx, t.Format(suf)}, ".")
				return strings.ToLower(tmFormat)
			}
			data := make(map[string]interface{})
			for k, v := range op.Data {
				data[k] = v
			}
			data["_source_id"] = objectID
			if ic.config.IndexOplogTime == false {
				secs := int64(op.Timestamp.T)
				t := time.Unix(secs, 0).UTC()
				data[ic.config.OplogTsFieldName] = op.Timestamp
				data[ic.config.OplogDateFieldName] = t.Format(ic.config.OplogDateFieldFormat)
			}
			req := elastic.NewBulkIndexRequest()
			req.UseEasyJSON(ic.config.EnableEasyJSON)
			req.Index(tmIndex(indexType.Index))
			req.Pipeline(indexType.Pipeline)
			req.Routing(objectID)
			req.Doc(data)
			if meta.Index != "" {
				req.Index(tmIndex(meta.Index))
			}
			if meta.Pipeline != "" {
				req.Pipeline(meta.Pipeline)
			}
			if ingestAttachment {
				req.Pipeline("attachment")
			}
			if _, err = req.Source(); err == nil {
				ic.bulk.Add(req)
			}
		}
	}
	return
}

func (ic *indexClient) doIndex(op *gtm.Op) (err error) {
	if err = ic.mapData(op); err == nil {
		if op.Data != nil {
			err = ic.doIndexing(op)
		} else if op.IsUpdate() {
			ic.doDelete(op)
		}
	}
	return
}

func (ic *indexClient) runProcessor(op *gtm.Op) (err error) {
	input := &monstachemap.ProcessPluginInput{
		ElasticClient:        ic.client,
		ElasticBulkProcessor: ic.bulk,
		Timestamp:            op.Timestamp,
	}
	input.Document = op.Data
	if op.IsDelete() {
		input.Document = map[string]interface{}{
			"_id": op.Id,
		}
	}
	input.Namespace = op.Namespace
	input.Database = op.GetDatabase()
	input.Collection = op.GetCollection()
	input.Operation = op.Operation
	input.MongoClient = ic.mongo
	input.UpdateDescription = op.UpdateDescription
	err = processPlugin(input)
	return
}

func (ic *indexClient) routeProcess(op *gtm.Op) (err error) {
	rop := &gtm.Op{
		Id:                op.Id,
		Operation:         op.Operation,
		Namespace:         op.Namespace,
		Source:            op.Source,
		Timestamp:         op.Timestamp,
		UpdateDescription: op.UpdateDescription,
	}
	if op.Data != nil {
		var data []byte
		data, err = bson.Marshal(op.Data)
		if err == nil {
			var m map[string]interface{}
			err = bson.Unmarshal(data, &m)
			if err == nil {
				rop.Data = m
			}
		}
	}
	ic.processC <- rop
	return
}

func (ic *indexClient) routeDrop(op *gtm.Op) (err error) {
	ic.bulk.Flush()
	err = ic.doDrop(op)
	return
}

func (ic *indexClient) routeDeleteRelate(op *gtm.Op) (err error) {
	if rs := relates[op.Namespace]; len(rs) != 0 {
		var delData map[string]interface{}
		useFind := false
		for _, r := range rs {
			if r.SrcField != "_id" {
				useFind = true
				break
			}
		}
		if useFind {
			delData = ic.findDeletedSrcDoc(op)
		} else {
			delData = map[string]interface{}{
				"_id": op.Id,
			}
		}
		if delData != nil {
			rop := &gtm.Op{
				Id:        op.Id,
				Operation: op.Operation,
				Namespace: op.Namespace,
				Source:    op.Source,
				Timestamp: op.Timestamp,
				Data:      delData,
			}
			select {
			case ic.relateC <- rop:
			default:
				errorLog.Printf(relateQueueOverloadMsg, rop.Namespace, rop.Id)
			}
		}
	}
	return

}

func (ic *indexClient) routeDelete(op *gtm.Op) (err error) {
	if len(ic.config.Relate) > 0 {
		err = ic.routeDeleteRelate(op)
	}
	ic.doDelete(op)
	return
}

func (ic *indexClient) routeDataRelate(op *gtm.Op) (skip bool, err error) {
	rs := relates[op.Namespace]
	if len(rs) == 0 {
		return
	}
	skip = true
	for _, r := range rs {
		if r.KeepSrc {
			skip = false
			break
		}
	}
	if skip {
		select {
		case ic.relateC <- op:
		default:
			errorLog.Printf(relateQueueOverloadMsg, op.Namespace, op.Id)
		}
	} else {
		rop := &gtm.Op{
			Id:                op.Id,
			Operation:         op.Operation,
			Namespace:         op.Namespace,
			Source:            op.Source,
			Timestamp:         op.Timestamp,
			UpdateDescription: op.UpdateDescription,
		}
		var data []byte
		data, err = bson.Marshal(op.Data)
		if err == nil {
			var m map[string]interface{}
			err = bson.Unmarshal(data, &m)
			if err == nil {
				rop.Data = m
			}
		}
		select {
		case ic.relateC <- rop:
		default:
			errorLog.Printf(relateQueueOverloadMsg, rop.Namespace, rop.Id)
		}
	}
	return
}

func (ic *indexClient) routeData(op *gtm.Op) (err error) {
	skip := false
	if op.IsSourceOplog() && len(ic.config.Relate) > 0 {
		skip, err = ic.routeDataRelate(op)
	}
	if !skip {
		if ic.hasFileContent(op) {
			ic.fileC <- op
		} else {
			ic.indexC <- op
		}
	}
	return
}

func (ic *indexClient) routeOp(op *gtm.Op) (err error) {
	if processPlugin != nil {
		err = ic.routeProcess(op)
	}
	if op.IsDrop() {
		err = ic.routeDrop(op)
	} else if op.IsDelete() {
		err = ic.routeDelete(op)
	} else if op.Data != nil {
		err = ic.routeData(op)
	}
	return
}

func (ic *indexClient) processErr(err error) {
	config := ic.config
	mux.Lock()
	defer mux.Unlock()
	exitStatus = 1
	var ee *elastic.Error
	if errors.As(err, &ee) {
		edata, _ := json.Marshal(ee.Details)
		errorLog.Printf("%s: [details=%s]\n", err, edata)
	} else {
		errorLog.Println(err)
	}
	if config.FailFast {
		os.Exit(exitStatus)
	}
}

func (ic *indexClient) doIndexStats() (err error) {
	var hostname string
	doc := make(map[string]interface{})
	t := time.Now().UTC()
	doc["Timestamp"] = t.Format("2006-01-02T15:04:05")
	hostname, err = os.Hostname()
	if err == nil {
		doc["Host"] = hostname
	}
	doc["Pid"] = os.Getpid()
	doc["Stats"] = ic.bulk.Stats()
	index := strings.ToLower(t.Format(ic.config.StatsIndexFormat))
	req := elastic.NewBulkIndexRequest().Index(index)
	req.UseEasyJSON(ic.config.EnableEasyJSON)
	req.Doc(doc)
	ic.bulkStats.Add(req)
	return
}

func (ic *indexClient) dropDBMeta(db string) (err error) {
	if ic.config.DeleteStrategy == statefulDeleteStrategy {
		col := ic.mongo.Database(ic.config.ConfigDatabaseName).Collection("meta")
		q := bson.M{"db": db}
		_, err = col.DeleteMany(context.Background(), q)
	}
	return
}

func (ic *indexClient) dropCollectionMeta(namespace string) (err error) {
	if ic.config.DeleteStrategy == statefulDeleteStrategy {
		col := ic.mongo.Database(ic.config.ConfigDatabaseName).Collection("meta")
		q := bson.M{"namespace": namespace}
		_, err = col.DeleteMany(context.Background(), q)
	}
	return
}

func (meta *indexingMeta) load(metaAttrs map[string]interface{}) {
	var v interface{}
	var ok bool
	var s string
	if _, ok = metaAttrs["skip"]; ok {
		meta.Skip = true
	}
	if v, ok = metaAttrs["routing"]; ok {
		meta.Routing = fmt.Sprintf("%v", v)
	}
	if v, ok = metaAttrs["index"]; ok {
		meta.Index = fmt.Sprintf("%v", v)
	}
	if v, ok = metaAttrs["id"]; ok {
		op := &gtm.Op{
			Id: v,
		}
		meta.ID = opIDToString(op)
	}
	if v, ok = metaAttrs["type"]; ok {
		meta.Type = fmt.Sprintf("%v", v)
	}
	if v, ok = metaAttrs["parent"]; ok {
		meta.Parent = fmt.Sprintf("%v", v)
	}
	if v, ok = metaAttrs["version"]; ok {
		s = fmt.Sprintf("%v", v)
		if version, err := strconv.ParseInt(s, 10, 64); err == nil {
			meta.Version = version
		} else {
			errorLog.Printf("Error applying version metadata: %s", err)
		}
	}
	if v, ok = metaAttrs["versionType"]; ok {
		meta.VersionType = fmt.Sprintf("%v", v)
	}
	if v, ok = metaAttrs["pipeline"]; ok {
		meta.Pipeline = fmt.Sprintf("%v", v)
	}
	if v, ok = metaAttrs["retryOnConflict"]; ok {
		s = fmt.Sprintf("%v", v)
		if roc, err := strconv.Atoi(s); err == nil {
			meta.RetryOnConflict = roc
		} else {
			errorLog.Printf("Error applying retryOnConflict metadata: %s", err)
		}
	}
}

func (meta *indexingMeta) shouldSave(config *configOptions) bool {
	if config.DeleteStrategy == statefulDeleteStrategy {
		return (meta.Routing != "" ||
			meta.Index != "" ||
			meta.Type != "" ||
			meta.Parent != "" ||
			meta.Pipeline != "")
	}
	return false
}

func (ic *indexClient) setIndexMeta(namespace, id string, meta *indexingMeta) error {
	config := ic.config
	col := ic.mongo.Database(config.ConfigDatabaseName).Collection("meta")
	metaID := fmt.Sprintf("%s.%s", namespace, id)
	doc := map[string]interface{}{
		"id":        meta.ID,
		"routing":   meta.Routing,
		"index":     meta.Index,
		"type":      meta.Type,
		"parent":    meta.Parent,
		"pipeline":  meta.Pipeline,
		"db":        strings.SplitN(namespace, ".", 2)[0],
		"namespace": namespace,
	}
	opts := options.Update()
	opts.SetUpsert(true)
	_, err := col.UpdateOne(context.Background(), bson.M{
		"_id": metaID,
	}, bson.M{
		"$set": doc,
	}, opts)
	return err
}

func (ic *indexClient) getIndexMeta(namespace, id string) (meta *indexingMeta) {
	meta = &indexingMeta{}
	config := ic.config
	col := ic.mongo.Database(config.ConfigDatabaseName).Collection("meta")
	metaID := fmt.Sprintf("%s.%s", namespace, id)
	result := col.FindOne(context.Background(), bson.M{
		"_id": metaID,
	})
	if err := result.Err(); err == nil {
		doc := make(map[string]interface{})
		if err = result.Decode(&doc); err == nil {
			if doc["id"] != nil {
				meta.ID = doc["id"].(string)
			}
			if doc["routing"] != nil {
				meta.Routing = doc["routing"].(string)
			}
			if doc["index"] != nil {
				meta.Index = strings.ToLower(doc["index"].(string))
			}
			if doc["type"] != nil {
				meta.Type = doc["type"].(string)
			}
			if doc["parent"] != nil {
				meta.Parent = doc["parent"].(string)
			}
			if doc["pipeline"] != nil {
				meta.Pipeline = doc["pipeline"].(string)
			}
			col.DeleteOne(context.Background(), bson.M{"_id": metaID})
		}
	}
	return
}

func loadBuiltinFunctions(client *mongo.Client, config *configOptions) {
	scriptEnvMaps := []map[string]*executionEnv{mapEnvs, filterEnvs}
	loadBuiltinFunctionsForEnvs(scriptEnvMaps, client, config)
}

func loadBuiltinFunctionsForEnvs(envMaps []map[string]*executionEnv, client *mongo.Client, config *configOptions) {
	for _, envMap := range envMaps {
		for ns, env := range envMap {
			var fa *findConf
			fa = &findConf{
				client: client,
				name:   "findId",
				vm:     env.VM,
				ns:     ns,
				byID:   true,
			}
			if err := env.VM.Set(fa.name, makeFind(fa)); err != nil {
				errorLog.Fatalln(err)
			}
			fa = &findConf{
				client: client,
				name:   "findOne",
				vm:     env.VM,
				ns:     ns,
			}
			if err := env.VM.Set(fa.name, makeFind(fa)); err != nil {
				errorLog.Fatalln(err)
			}
			fa = &findConf{
				client: client,
				name:   "find",
				vm:     env.VM,
				ns:     ns,
				multi:  true,
			}
			if err := env.VM.Set(fa.name, makeFind(fa)); err != nil {
				errorLog.Fatalln(err)
			}
			fa = &findConf{
				client:        client,
				name:          "pipe",
				vm:            env.VM,
				ns:            ns,
				multi:         true,
				pipe:          true,
				pipeAllowDisk: config.PipeAllowDisk,
			}
			if err := env.VM.Set(fa.name, makeFind(fa)); err != nil {
				errorLog.Fatalln(err)
			}
		}
	}
}

func (fc *findCall) setDatabase(topts map[string]interface{}) (err error) {
	if ov, ok := topts["database"]; ok {
		if ovs, ok := ov.(string); ok {
			fc.db = ovs
		} else {
			err = errors.New("Invalid database option value")
		}
	}
	return
}

func (fc *findCall) setCollection(topts map[string]interface{}) (err error) {
	if ov, ok := topts["collection"]; ok {
		if ovs, ok := ov.(string); ok {
			fc.col = ovs
		} else {
			err = errors.New("Invalid collection option value")
		}
	}
	return
}

func (fc *findCall) setSelect(topts map[string]interface{}) (err error) {
	if ov, ok := topts["select"]; ok {
		if ovsel, ok := ov.(map[string]interface{}); ok {
			for k, v := range ovsel {
				if vi, ok := v.(int64); ok {
					fc.sel[k] = int(vi)
				}
			}
		} else {
			err = errors.New("Invalid select option value")
		}
	}
	return
}

func (fc *findCall) setSort(topts map[string]interface{}) (err error) {
	if ov, ok := topts["sort"]; ok {
		if ovsort, ok := ov.(map[string]interface{}); ok {
			for k, v := range ovsort {
				if vi, ok := v.(int64); ok {
					fc.sort[k] = int(vi)
				}
			}
		} else {
			err = errors.New("Invalid sort option value")
		}
		fc.setSort(map[string]interface{}{"joe": "rick"})
	}
	return
}

func (fc *findCall) setLimit(topts map[string]interface{}) (err error) {
	if ov, ok := topts["limit"]; ok {
		if ovl, ok := ov.(int64); ok {
			fc.limit = int(ovl)
		} else {
			err = errors.New("Invalid limit option value")
		}
	}
	return
}

func (fc *findCall) setQuery(v otto.Value) (err error) {
	var q interface{}
	if q, err = v.Export(); err == nil {
		fc.query = fc.restoreIds(deepExportValue(q))
	}
	return
}

func (fc *findCall) setOptions(v otto.Value) (err error) {
	var opts interface{}
	if opts, err = v.Export(); err == nil {
		switch topts := opts.(type) {
		case map[string]interface{}:
			if err = fc.setDatabase(topts); err != nil {
				return
			}
			if err = fc.setCollection(topts); err != nil {
				return
			}
			if err = fc.setSelect(topts); err != nil {
				return
			}
			if fc.isMulti() {
				if err = fc.setSort(topts); err != nil {
					return
				}
				if err = fc.setLimit(topts); err != nil {
					return
				}
			}
		default:
			err = errors.New("Invalid options argument")
			return
		}
	} else {
		err = errors.New("Invalid options argument")
	}
	return
}

func (fc *findCall) setDefaults() {
	if fc.config.ns != "" {
		ns := strings.SplitN(fc.config.ns, ".", 2)
		fc.db = ns[0]
		fc.col = ns[1]
	}
}

func (fc *findCall) getCollection() *mongo.Collection {
	return fc.client.Database(fc.db).Collection(fc.col)
}

func (fc *findCall) getVM() *otto.Otto {
	return fc.config.vm
}

func (fc *findCall) getFunctionName() string {
	return fc.config.name
}

func (fc *findCall) isMulti() bool {
	return fc.config.multi
}

func (fc *findCall) isPipe() bool {
	return fc.config.pipe
}

func (fc *findCall) pipeAllowDisk() bool {
	return fc.config.pipeAllowDisk
}

func (fc *findCall) logError(err error) {
	errorLog.Printf("Error in function %s: %s\n", fc.getFunctionName(), err)
}

func (fc *findCall) restoreIds(v interface{}) (r interface{}) {
	switch vt := v.(type) {
	case string:
		if oi, err := primitive.ObjectIDFromHex(vt); err == nil {
			r = oi
		} else {
			r = v
		}
	case []map[string]interface{}:
		var avs []interface{}
		for _, av := range vt {
			mvs := make(map[string]interface{})
			for k, v := range av {
				mvs[k] = fc.restoreIds(v)
			}
			avs = append(avs, mvs)
		}
		r = avs
	case []interface{}:
		var avs []interface{}
		for _, av := range vt {
			avs = append(avs, fc.restoreIds(av))
		}
		r = avs
	case map[string]interface{}:
		mvs := make(map[string]interface{})
		for k, v := range vt {
			mvs[k] = fc.restoreIds(v)
		}
		r = mvs
	default:
		r = v
	}
	return
}

func (fc *findCall) execute() (r otto.Value, err error) {
	var cursor *mongo.Cursor
	col := fc.getCollection()
	query := fc.query
	if fc.isMulti() {
		if fc.isPipe() {
			ao := options.Aggregate()
			ao.SetAllowDiskUse(fc.pipeAllowDisk())
			cursor, err = col.Aggregate(context.Background(), query, ao)
			if err != nil {
				return
			}
		} else {
			fo := options.Find()
			if fc.limit > 0 {
				fo.SetLimit(int64(fc.limit))
			}
			if len(fc.sort) > 0 {
				fo.SetSort(fc.sort)
			}
			if len(fc.sel) > 0 {
				fo.SetProjection(fc.sel)
			}
			cursor, err = col.Find(context.Background(), query, fo)
			if err != nil {
				return
			}
		}
		var rdocs []map[string]interface{}
		for cursor.Next(context.Background()) {
			doc := make(map[string]interface{})
			if err = cursor.Decode(&doc); err != nil {
				return
			}
			rdocs = append(rdocs, convertMapJavascript(doc))
		}
		r, err = fc.getVM().ToValue(rdocs)
	} else {
		fo := options.FindOne()
		if fc.config.byID {
			query = bson.M{"_id": query}
		}
		if len(fc.sel) > 0 {
			fo.SetProjection(fc.sel)
		}
		result := col.FindOne(context.Background(), query, fo)
		if err = result.Err(); err == nil {
			doc := make(map[string]interface{})
			if err = result.Decode(&doc); err == nil {
				rdoc := convertMapJavascript(doc)
				r, err = fc.getVM().ToValue(rdoc)
			}
		}
	}
	return
}

func makeFind(fa *findConf) func(otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) (r otto.Value) {
		var err error
		fc := &findCall{
			config: fa,
			client: fa.client,
			sort:   make(map[string]int),
			sel:    make(map[string]int),
		}
		fc.setDefaults()
		args := call.ArgumentList
		argLen := len(args)
		r = otto.NullValue()
		if argLen >= 1 {
			if argLen >= 2 {
				if err = fc.setOptions(call.Argument(1)); err != nil {
					fc.logError(err)
					return
				}
			}
			if fc.db == "" || fc.col == "" {
				fc.logError(errors.New("Find call must specify db and collection"))
				return
			}
			if err = fc.setQuery(call.Argument(0)); err == nil {
				var result otto.Value
				if result, err = fc.execute(); err == nil {
					r = result
				} else if err != mongo.ErrNoDocuments {
					fc.logError(err)
				}
			} else {
				fc.logError(err)
			}
		} else {
			fc.logError(errors.New("At least one argument is required"))
		}
		return
	}
}

func (ic *indexClient) findDeletedSrcDoc(op *gtm.Op) map[string]interface{} {
	objectID := opIDToString(op)
	termQuery := elastic.NewTermQuery("_id", objectID)
	search := ic.client.Search()
	search.Size(1)
	search.Index(ic.config.DeleteIndexPattern)
	search.Query(termQuery)
	searchResult, err := search.Do(context.Background())
	if err != nil {
		errorLog.Printf("Unable to find deleted document %s: %s", objectID, err)
		return nil
	}
	if searchResult.Hits == nil {
		errorLog.Printf("Unable to find deleted document %s", objectID)
		return nil
	}
	if searchResult.TotalHits() == 0 {
		errorLog.Printf("Found no hits for deleted document %s", objectID)
		return nil
	}
	if searchResult.TotalHits() > 1 {
		errorLog.Printf("Found multiple hits for deleted document %s", objectID)
		return nil
	}
	hit := searchResult.Hits.Hits[0]
	if hit.Source == nil {
		errorLog.Printf("Source unavailable for deleted document %s", objectID)
		return nil
	}
	var src map[string]interface{}
	if err = json.Unmarshal(hit.Source, &src); err == nil {
		src["_id"] = op.Id
		return src
	}
	errorLog.Printf("Unable to unmarshal deleted document %s: %s", objectID, err)
	return nil
}

func tsVersion(ts primitive.Timestamp) int64 {
	t, i := int64(ts.T), int64(ts.I)
	version := (t << 32) | i
	return version
}

func (ic *indexClient) doDelete(op *gtm.Op) {
	req := elastic.NewBulkDeleteRequest()
	req.UseEasyJSON(ic.config.EnableEasyJSON)
	if ic.config.DeleteStrategy == ignoreDeleteStrategy {
		return
	}
	objectID, indexType, meta := opIDToString(op), ic.mapIndex(op), &indexingMeta{}
	if objectID == "" {
		errorLog.Println("Unable to delete document due to empty _id value")
		return
	}
	req.Id(objectID)
	if ic.config.IndexAsUpdate == false {
		req.Version(tsVersion(op.Timestamp))
		req.VersionType("external")
	}
	if ic.config.DeleteStrategy == statefulDeleteStrategy {
		if routingNamespaces[""] || routingNamespaces[op.Namespace] {
			meta = ic.getIndexMeta(op.Namespace, objectID)
		}
		req.Index(indexType.Index)
		if meta.Index != "" {
			req.Index(meta.Index)
		}
		if meta.Routing != "" {
			req.Routing(meta.Routing)
		}
		if meta.Parent != "" {
			req.Parent(meta.Parent)
		}
	} else if ic.config.DeleteStrategy == statelessDeleteStrategy {
		if routingNamespaces[""] || routingNamespaces[op.Namespace] {
			termQuery := elastic.NewTermQuery("_id", objectID)
			search := ic.client.Search()
			search.FetchSource(false)
			search.Size(1)
			search.Index(ic.config.DeleteIndexPattern)
			search.Query(termQuery)
			searchResult, err := search.Do(context.Background())
			if err != nil {
				errorLog.Printf("Unable to delete document %s: %s",
					objectID, err)
				return
			}
			if searchResult.Hits != nil && searchResult.TotalHits() == 1 {
				hit := searchResult.Hits.Hits[0]
				req.Index(hit.Index)
				if hit.Routing != "" {
					req.Routing(hit.Routing)
				}
				if hit.Parent != "" {
					req.Parent(hit.Parent)
				}
			} else {
				errorLog.Printf("Failed to find unique document %s for deletion using index pattern %s",
					objectID, ic.config.DeleteIndexPattern)
				return
			}
		} else {
			req.Index(indexType.Index)
		}
	} else {
		return
	}
	ic.bulk.Add(req)
	return
}

func logRotateDefaults() logRotate {
	return logRotate{
		MaxSize:    500, //megabytes
		MaxAge:     28,  // days
		MaxBackups: 5,
		LocalTime:  false,
		Compress:   false,
	}
}

func gtmDefaultSettings() gtmSettings {
	return gtmSettings{
		ChannelSize:    gtmChannelSizeDefault,
		BufferSize:     32,
		BufferDuration: "75ms",
		MaxAwaitTime:   "",
	}
}

func (ic *indexClient) notifySdFailed(err error) {
	if err != nil {
		errorLog.Printf("Systemd notification failed: %s", err)
	} else {
		if ic.config.Verbose {
			warnLog.Println("Systemd notification not supported (i.e. NOTIFY_SOCKET is unset)")
		}
	}
}

func (ic *indexClient) watchdogSdFailed(err error) {
	if err != nil {
		errorLog.Printf("Error determining systemd WATCHDOG interval: %s", err)
	} else {
		if ic.config.Verbose {
			warnLog.Println("Systemd WATCHDOG not enabled")
		}
	}
}

func (ctx *httpServerCtx) serveHTTP() {
	s := ctx.httpServer
	if ctx.config.Verbose {
		infoLog.Printf("Starting http server at %s", s.Addr)
	}
	ctx.started = time.Now()
	err := s.ListenAndServe()
	if !ctx.shutdown {
		errorLog.Fatalf("Unable to serve http at address %s: %s", s.Addr, err)
	}
}

func (ctx *httpServerCtx) buildServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/started", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		data := (time.Now().Sub(ctx.started)).String()
		w.Write([]byte(data))
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	if ctx.config.Stats {
		mux.HandleFunc("/stats", func(w http.ResponseWriter, req *http.Request) {
			stats, err := json.MarshalIndent(ctx.bulk.Stats(), "", "    ")
			if err == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(200)
				w.Write(stats)
				fmt.Fprintln(w)
			} else {
				w.WriteHeader(500)
				fmt.Fprintf(w, "Unable to print statistics: %s", err)
			}
		})
	}
	mux.HandleFunc("/instance", func(w http.ResponseWriter, req *http.Request) {
		hostname, err := os.Hostname()
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Unable to get hostname for instance info: %s", err)
			return
		}
		status := instanceStatus{
			Pid:         os.Getpid(),
			Hostname:    hostname,
			ResumeName:  ctx.config.ResumeName,
			ClusterName: ctx.config.ClusterName,
		}
		respC := make(chan *statusResponse)
		statusReq := &statusRequest{
			responseC: respC,
		}
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		select {
		case ctx.statusReqC <- statusReq:
			srsp := <-respC
			if srsp != nil {
				status.Enabled = srsp.enabled
				status.LastTs = srsp.lastTs
				if srsp.lastTs.T != 0 {
					status.LastTsFormat = time.Unix(int64(srsp.lastTs.T), 0).Format("2006-01-02T15:04:05")
				}
			}
			data, err := json.Marshal(status)
			if err != nil {
				w.WriteHeader(500)
				fmt.Fprintf(w, "Unable to print instance info: %s", err)
				break
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write(data)
			fmt.Fprintln(w)
			break
		case <-timer.C:
			w.WriteHeader(500)
			fmt.Fprintf(w, "Timeout getting instance info")
			break
		}
	})
	if ctx.config.Pprof {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
	s := &http.Server{
		Addr:     ctx.config.HTTPServerAddr,
		Handler:  mux,
		ErrorLog: errorLog,
	}
	ctx.httpServer = s
}

func (ic *indexClient) startNotify() {
	go ic.notifySd()
}

func (ic *indexClient) notifySd() {
	var interval time.Duration
	config := ic.config
	if config.Verbose {
		infoLog.Println("Sending systemd READY=1")
	}
	sent, err := daemon.SdNotify(false, "READY=1")
	if sent {
		if config.Verbose {
			infoLog.Println("READY=1 successfully sent to systemd")
		}
	} else {
		ic.notifySdFailed(err)
		return
	}
	interval, err = daemon.SdWatchdogEnabled(false)
	if err != nil || interval == 0 {
		ic.watchdogSdFailed(err)
		return
	}
	for {
		if config.Verbose {
			infoLog.Println("Sending systemd WATCHDOG=1")
		}
		sent, err = daemon.SdNotify(false, "WATCHDOG=1")
		if sent {
			if config.Verbose {
				infoLog.Println("WATCHDOG=1 successfully sent to systemd")
			}
		} else {
			ic.notifySdFailed(err)
			return
		}
		time.Sleep(interval / 2)
	}
}

func (config *configOptions) makeShardInsertHandler() gtm.ShardInsertHandler {
	return func(shardInfo *gtm.ShardInfo) (*mongo.Client, error) {
		shardURL := shardInfo.GetURL()
		infoLog.Printf("Adding shard found at %s\n", cleanMongoURL(shardURL))
		return config.dialMongo(shardURL)
	}
}

func buildPipe(config *configOptions) func(string, bool) ([]interface{}, error) {
	if pipePlugin != nil {
		return pipePlugin
	} else if len(pipeEnvs) > 0 {
		return func(ns string, changeEvent bool) ([]interface{}, error) {
			mux.Lock()
			defer mux.Unlock()
			nss := []string{"", ns}
			for _, ns := range nss {
				if env := pipeEnvs[ns]; env != nil {
					env.lock.Lock()
					defer env.lock.Unlock()
					val, err := env.VM.Call("module.exports", ns, ns, changeEvent)
					if err != nil {
						return nil, err
					}
					if strings.ToLower(val.Class()) == "array" {
						data, err := val.Export()
						if err != nil {
							return nil, err
						} else if data == val {
							return nil, errors.New("Exported pipeline function must return an array")
						} else {
							switch data.(type) {
							case []map[string]interface{}:
								ds := data.([]map[string]interface{})
								var is []interface{} = make([]interface{}, len(ds))
								for i, d := range ds {
									is[i] = deepExportValue(d)
								}
								return is, nil
							case []interface{}:
								ds := data.([]interface{})
								if len(ds) > 0 {
									errorLog.Fatalln("Pipeline function must return an array of objects")
								}
								return nil, nil
							default:
								errorLog.Fatalln("Pipeline function must return an array of objects")
							}
						}
					} else {
						return nil, errors.New("Exported pipeline function must return an array")
					}
				}
			}
			return nil, nil
		}
	}
	return nil
}

func (sh *sigHandler) start() {
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
		select {
		case <-sigs:
			// we never got started so simply exit
			os.Exit(0)
		case ic := <-sh.clientStartedC:
			<-sigs
			go func() {
				// forced shutdown on 2nd signal
				<-sigs
				infoLog.Println("Forcing shutdown, bye bye...")
				os.Exit(1)
			}()
			// we started processing events so do a clean shutdown
			ic.stopAllWorkers()
			ic.doneC <- 10
		}
	}()
}

func (ic *indexClient) startHTTPServer() {
	config := ic.config
	if config.EnableHTTPServer {
		ic.hsc = &httpServerCtx{
			bulk:       ic.bulk,
			config:     ic.config,
			statusReqC: ic.statusReqC,
		}
		ic.hsc.buildServer()
		go ic.hsc.serveHTTP()
	}
}

func (ic *indexClient) setupFileIndexing() {
	config := ic.config
	if config.IndexFiles {
		if len(config.FileNamespaces) == 0 {
			errorLog.Fatalln("File indexing is ON but no file namespaces are configured")
		}
		if err := ic.ensureFileMapping(); err != nil {
			errorLog.Fatalf("Unable to setup file indexing: %s", err)
		}
	}
}

func (ic *indexClient) setupBulk() {
	config := ic.config
	bulk, err := config.newBulkProcessor(ic.client)
	if err != nil {
		errorLog.Fatalf("Unable to start bulk processor: %s", err)
	}
	var bulkStats *elastic.BulkProcessor
	if config.IndexStats {
		bulkStats, err = config.newStatsBulkProcessor(ic.client)
		if err != nil {
			errorLog.Fatalf("Unable to start stats bulk processor: %s", err)
		}
	}
	ic.bulk = bulk
	ic.bulkStats = bulkStats
}

// setupPlugin call the 'Init' function in plugin while it's not nil after
// the mongo/elastic setup actions with initialized mongo/elastic clients.
func (ic *indexClient) setupPlugin() {
	if initPlugin == nil {
		return
	}

	// check elastic client/bulk.
	if ic.client == nil || ic.bulk == nil {
		errorLog.Fatalf("Unable to setup plugin: elastic client not initialized")
	}

	// check mongo client.
	if ic.mongo == nil {
		errorLog.Fatalf("Unable to setup plugin: mongo client not initialized")
	}

	// input of plugin 'Init' function.
	input := monstachemap.InitPluginInput{
		ElasticClient:        ic.client,
		ElasticBulkProcessor: ic.bulk,
		MongoClient:          ic.mongo,
	}

	// call the 'Init' function now.
	if err := initPlugin(&input); err != nil {
		errorLog.Fatalf("Unable to setup plugin: %s", err)
	}
}

func (ic *indexClient) run() {
	ic.startNotify()
	ic.setupFileIndexing()
	ic.setupBulk()
	ic.setupPlugin()
	ic.startHTTPServer()
	ic.startCluster()
	ic.startRelate()
	ic.startIndex()
	ic.startDownload()
	ic.startPostProcess()
	ic.clusterWait()
	ic.startListen()
	ic.startReadWait()
	ic.startExpireCreds()
	ic.eventLoop()
}

func (ic *indexClient) startDownload() {
	for i := 0; i < ic.config.FileDownloaders; i++ {
		ic.fileWg.Add(1)
		go func() {
			defer ic.fileWg.Done()
			for op := range ic.fileC {
				if err := ic.addFileContent(op); err != nil {
					ic.processErr(err)
				}
				ic.indexC <- op
			}
		}()
	}
}

func (ic *indexClient) startPostProcess() {
	for i := 0; i < ic.config.PostProcessors; i++ {
		ic.processWg.Add(1)
		go func() {
			defer ic.processWg.Done()
			for op := range ic.processC {
				if err := ic.runProcessor(op); err != nil {
					ic.processErr(err)
				}
			}
		}()
	}
}

func (ic *indexClient) stopAllWorkers() {
	infoLog.Println("Stopping all workers")
	ic.gtmCtx.Stop()
	<-ic.opsConsumed
	close(ic.relateC)
	ic.relateWg.Wait()
	close(ic.fileC)
	ic.fileWg.Wait()
	close(ic.indexC)
	ic.indexWg.Wait()
	close(ic.processC)
	ic.processWg.Wait()
}

func (ic *indexClient) startReadWait() {
	if len(ic.config.DirectReadNs) > 0 || ic.config.ExitAfterDirectReads {
		go func() {
			ic.gtmCtx.DirectReadWg.Wait()
			infoLog.Println("Direct reads completed")
			if ic.config.Resume {
				ic.saveTimestampFromReplStatus()
			}
			if ic.config.DirectReadStateful && len(ic.config.DirectReadNs) > 0 {
				if err := ic.saveDirectReadNamespaces(); err != nil {
					errorLog.Printf("Error saving direct read state: %s", err)
				}
			}
			if ic.config.ExitAfterDirectReads {
				ic.stopAllWorkers()
				ic.doneC <- 30
			}
		}()
	}
}

func (ic *indexClient) startExpireCreds() {
	conf := ic.config
	ac := conf.AWSConnect
	if ac.forceExpireCreds() {
		duration, err := time.ParseDuration(ac.ForceExpire)
		if err != nil {
			errorLog.Fatalf("Error starting credential expiry: %s", err)
			return
		}
		go func() {
			ticker := time.NewTicker(duration)
			defer ticker.Stop()
			for range ticker.C {
				ac.creds.Expire()
				infoLog.Println("Force expired AWS credential")
			}
		}()
	}
	if ac.watchCreds() {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			errorLog.Fatalf("Error starting file watcher: %s", err)
		}
		go func() {
			defer watcher.Close()
			for {
				select {
				case _, ok := <-watcher.Events:
					if !ok {
						return
					}
					ac.creds.Expire()
					infoLog.Println("File watcher expired AWS credential")
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					errorLog.Printf("Error watching path %s: %s", ac.watchFilePath(), err)
				}
			}
		}()
		err = watcher.Add(ac.watchFilePath())
		if err != nil {
			errorLog.Fatalf("Error adding file watcher for path %s: %s", ac.watchFilePath(), err)
		}
	}
}

func (ic *indexClient) dialShards() []*mongo.Client {
	var mongos []*mongo.Client
	// get the list of shard servers
	shardInfos := gtm.GetShards(ic.mongoConfig)
	if len(shardInfos) == 0 {
		errorLog.Fatalln("Shards enabled but none found in config.shards collection")
	}
	// add each shard server to the sync list
	for _, shardInfo := range shardInfos {
		shardURL := shardInfo.GetURL()
		infoLog.Printf("Adding shard found at %s\n", cleanMongoURL(shardURL))
		shard, err := ic.config.dialMongo(shardURL)
		if err != nil {
			errorLog.Fatalf("Unable to connect to mongodb shard using URL %s: %s", cleanMongoURL(shardURL), err)
		}
		mongos = append(mongos, shard)
	}
	return mongos
}

func (ic *indexClient) buildTokenGen() gtm.ResumeTokenGenenerator {
	config := ic.config
	var token gtm.ResumeTokenGenenerator
	if !config.Resume || (config.ResumeStrategy != tokenResumeStrategy) {
		return token
	}
	token = func(client *mongo.Client, streamID string, options *gtm.Options) (interface{}, error) {
		var t interface{} = nil
		var err error
		col := client.Database(config.ConfigDatabaseName).Collection("tokens")
		result := col.FindOne(context.Background(), bson.M{
			"resumeName": config.ResumeName,
			"streamID":   streamID,
		})
		if err = result.Err(); err == nil {
			doc := make(map[string]interface{})
			if err = result.Decode(&doc); err == nil {
				t = doc["token"]
				if t != nil {
					infoLog.Printf("Resuming stream '%s' from collection %s.tokens using resume name '%s'",
						streamID, config.ConfigDatabaseName, config.ResumeName)
				}
			}
		}
		return t, err
	}
	return token
}

func (ic *indexClient) buildTimestampGen() gtm.TimestampGenerator {
	var after gtm.TimestampGenerator
	config := ic.config
	if config.ResumeStrategy != timestampResumeStrategy {
		return after
	}
	if config.Replay {
		after = func(client *mongo.Client, options *gtm.Options) (primitive.Timestamp, error) {
			ts, _ := gtm.FirstOpTimestamp(client, options)
			// add ten seconds as oldest items often fall off the oplog
			ts.T += 10
			ts.I = 0
			infoLog.Printf("Replaying from timestamp %+v", ts)
			return ts, nil
		}
	} else if config.ResumeFromTimestamp != 0 {
		after = func(client *mongo.Client, options *gtm.Options) (primitive.Timestamp, error) {
			return primitive.Timestamp{
				T: uint32(config.ResumeFromTimestamp >> 32),
				I: uint32(config.ResumeFromTimestamp),
			}, nil
		}
	} else if config.Resume {
		after = func(client *mongo.Client, options *gtm.Options) (primitive.Timestamp, error) {
			var candidateTs primitive.Timestamp
			var tsSource string
			var err error
			col := client.Database(config.ConfigDatabaseName).Collection("monstache")
			result := col.FindOne(context.Background(), bson.M{
				"_id": config.ResumeName,
			})
			if err = result.Err(); err == nil {
				doc := make(map[string]interface{})
				if err = result.Decode(&doc); err == nil {
					if doc["ts"] != nil {
						candidateTs = doc["ts"].(primitive.Timestamp)
						candidateTs.I++
						tsSource = oplog.TS_SOURCE_MONSTACHE
					}
				}
			}
			if candidateTs.T == 0 {
				candidateTs, _ = gtm.LastOpTimestamp(client, options)
				tsSource = oplog.TS_SOURCE_OPLOG
			}

			ts := <-ic.oplogTsResolver.GetResumeTimestamp(candidateTs, tsSource)
			infoLog.Printf("Resuming from timestamp %+v", ts)
			return ts, nil
		}
	}
	return after
}

func (ic *indexClient) buildConnections() []*mongo.Client {
	var mongos []*mongo.Client
	var err error
	config := ic.config
	if config.readShards() {
		// if we have a config server URL then we are running in a sharded cluster
		ic.mongoConfig, err = config.dialMongo(config.MongoConfigURL)
		if err != nil {
			errorLog.Fatalf("Unable to connect to mongodb config server using URL %s: %s",
				cleanMongoURL(config.MongoConfigURL), err)
		}
		mongos = ic.dialShards()
	} else {
		mongos = append(mongos, ic.mongo)
	}
	return mongos
}

func (ic *indexClient) buildFilterChain() []gtm.OpFilter {
	config := ic.config
	filterChain := []gtm.OpFilter{notMonstache(config), notSystem, notChunks}
	if config.readShards() {
		filterChain = append(filterChain, notConfig)
	}
	if config.NsRegex != "" {
		filterChain = append(filterChain, filterWithRegex(config.NsRegex))
	}
	if config.NsDropRegex != "" {
		filterChain = append(filterChain, filterDropWithRegex(config.NsDropRegex))
	}
	if config.NsExcludeRegex != "" {
		filterChain = append(filterChain, filterInverseWithRegex(config.NsExcludeRegex))
	}
	if config.NsDropExcludeRegex != "" {
		filterChain = append(filterChain, filterDropInverseWithRegex(config.NsDropExcludeRegex))
	}
	return filterChain
}

func (ic *indexClient) buildFilterArray() []gtm.OpFilter {
	config := ic.config
	filterArray := []gtm.OpFilter{}
	var pluginFilter gtm.OpFilter
	if config.Worker != "" {
		workerFilter, err := consistent.ConsistentHashFilter(config.Worker, config.Workers)
		if err != nil {
			errorLog.Fatalln(err)
		}
		filterArray = append(filterArray, workerFilter)
	} else if config.Workers != nil {
		errorLog.Fatalln("Workers configured but this worker is undefined. worker must be set to one of the workers.")
	}
	if filterPlugin != nil {
		pluginFilter = filterWithPlugin()
		filterArray = append(filterArray, pluginFilter)
	} else if len(filterEnvs) > 0 {
		pluginFilter = filterWithScript()
		filterArray = append(filterArray, pluginFilter)
	}
	if pluginFilter != nil {
		ic.filter = pluginFilter
	}
	return filterArray
}

func (ic *indexClient) buildDynamicDirectReadNs(filter gtm.OpFilter) (names []string) {
	client, config := ic.mongo, ic.config
	if config.DirectReadExcludeRegex != "" {
		filter = gtm.ChainOpFilters(filterInverseWithRegex(config.DirectReadExcludeRegex), filter)
	}
	if config.DirectReadIncludeRegex != "" {
		filter = gtm.ChainOpFilters(filterWithRegex(config.DirectReadIncludeRegex), filter)
	}

	dbs, err := client.ListDatabaseNames(context.Background(), bson.M{})
	if err != nil {
		errorLog.Fatalf("Failed to read database names for dynamic direct reads: %s", err)
	}
	for _, d := range dbs {
		if config.ignoreDatabaseForDirectReads(d) {
			continue
		}
		db := client.Database(d)
		cols, err := db.ListCollectionNames(context.Background(), bson.M{})
		if err != nil {
			errorLog.Fatalf("Failed to read db %s collection names for dynamic direct reads: %s", d, err)
			return
		}
		for _, c := range cols {
			if config.ignoreCollectionForDirectReads(c) {
				continue
			}
			ns := strings.Join([]string{d, c}, ".")
			if filter(&gtm.Op{Namespace: ns}) {
				names = append(names, ns)
			} else {
				infoLog.Printf("Excluding collection [%s] for dynamic direct reads", ns)
			}
		}
	}
	if len(names) == 0 {
		warnLog.Println("Dynamic direct read candidates: NONE")
	} else {
		infoLog.Printf("Dynamic direct read candidates: %v", names)
	}
	return
}

func (ic *indexClient) buildDynamicChangeStreamNs(filter gtm.OpFilter) (names []string) {
	client, config := ic.mongo, ic.config
	if config.DirectReadExcludeRegex != "" {
		filter = gtm.ChainOpFilters(filterInverseWithRegex(config.NsRegex), filter)
	}
	if config.DirectReadIncludeRegex != "" {
		filter = gtm.ChainOpFilters(filterWithRegex(config.NsRegex), filter)
	}

	dbs, err := client.ListDatabaseNames(context.Background(), bson.M{})
	if err != nil {
		errorLog.Fatalf("Failed to read database names for dynamic direct reads: %s", err)
	}

	uniqueNSMap := make(map[string]struct{}, 0)

	// has dynamic rules, watch database
	// at the same time match the exact table name as much as possible
	for _, d := range dbs {
		if config.ignoreDatabaseForChangeStreamReads(d) {
			continue
		}

		uniqueNSMap[d] = struct{}{}
		db := client.Database(d)
		cols, err := db.ListCollectionNames(context.Background(), bson.M{})
		if err != nil {
			errorLog.Fatalf("Failed to read db %s collection names for dynamic direct reads: %s", d, err)
			return
		}

		for _, c := range cols {
			if config.ignoreCollectionForChangeStreamReads(c) {
				continue
			}
			ns := strings.Join([]string{d, c}, ".")
			if filter(&gtm.Op{Namespace: ns}) {
				names = append(names, ns)
			} else {
				infoLog.Printf("Excluding collection [%s] for dynamic direct reads", ns)
			}
		}
	}

	// has dynamic rules, watch database
	for name := range uniqueNSMap {
		names = append(names, name)
	}

	if len(names) == 0 {
		warnLog.Println("Dynamic change stream read candidates: NONE")
	} else {
		infoLog.Printf("Dynamic change stream read candidates: %v", names)
	}
	return
}

func (ic *indexClient) parseBufferDuration() time.Duration {
	config := ic.config
	gtmBufferDuration, err := time.ParseDuration(config.GtmSettings.BufferDuration)
	if err != nil {
		errorLog.Fatalf("Unable to parse gtm buffer duration %s: %s",
			config.GtmSettings.BufferDuration, err)
	}
	return gtmBufferDuration
}

func (ic *indexClient) parseMaxAwaitTime() time.Duration {
	config := ic.config
	var maxAwaitTime time.Duration
	if config.GtmSettings.MaxAwaitTime != "" {
		var err error
		maxAwaitTime, err = time.ParseDuration(config.GtmSettings.MaxAwaitTime)
		if err != nil {
			errorLog.Fatalf("Unable to parse gtm max await time %s: %s",
				config.GtmSettings.MaxAwaitTime, err)

		}
	}
	return maxAwaitTime
}

func (ic *indexClient) buildGtmOptions() *gtm.Options {
	var nsFilter, filter, directReadFilter gtm.OpFilter
	config := ic.config
	filterChain := ic.buildFilterChain()
	filterArray := ic.buildFilterArray()
	nsFilter = gtm.ChainOpFilters(filterChain...)
	filter = gtm.ChainOpFilters(filterArray...)
	directReadFilter = gtm.ChainOpFilters(filterArray...)
	after := ic.buildTimestampGen()
	token := ic.buildTokenGen()

	if config.dynamicDirectReadList() {
		config.DirectReadNs = ic.buildDynamicDirectReadNs(nsFilter)
	}

	if config.DirectReadStateful {
		var err error
		config.DirectReadNs, err = ic.filterDirectReadNamespaces(config.DirectReadNs)
		if err != nil {
			errorLog.Fatalf("Error retrieving direct read state: %s", err)
		}
	}

	if config.dynamicChangeStreamList() {
		config.ChangeStreamNs = ic.buildDynamicChangeStreamNs(nsFilter)
	}

	gtmOpts := &gtm.Options{
		After:               after,
		Token:               token,
		Filter:              filter,
		NamespaceFilter:     nsFilter,
		OpLogDisabled:       config.EnableOplog == false,
		OpLogDatabaseName:   config.MongoOpLogDatabaseName,
		OpLogCollectionName: config.MongoOpLogCollectionName,
		ChannelSize:         config.GtmSettings.ChannelSize,
		Ordering:            gtm.AnyOrder,
		WorkerCount:         10,
		BufferDuration:      ic.parseBufferDuration(),
		BufferSize:          config.GtmSettings.BufferSize,
		DirectReadNs:        config.DirectReadNs,
		DirectReadSplitMax:  int32(config.DirectReadSplitMax),
		DirectReadConcur:    config.DirectReadConcur,
		DirectReadNoTimeout: config.DirectReadNoTimeout,
		DirectReadFilter:    directReadFilter,
		Log:                 infoLog,
		Pipe:                buildPipe(config),
		ChangeStreamNs:      config.ChangeStreamNs,
		DirectReadBounded:   config.DirectReadBounded,
		MaxAwaitTime:        ic.parseMaxAwaitTime(),
	}
	return gtmOpts
}

func (ic *indexClient) startListen() {
	config := ic.config
	conns := ic.buildConnections()

	if config.ResumeStrategy == timestampResumeStrategy {
		if config.ResumeFromEarliestTimestamp {
			ic.oplogTsResolver = oplog.NewTimestampResolverEarliest(len(conns), infoLog)
		} else {
			ic.oplogTsResolver = oplog.TimestampResolverSimple{}
		}
	}

	gtmOpts := ic.buildGtmOptions()
	ic.gtmCtx = gtm.StartMulti(conns, gtmOpts)
	if config.readShards() && !config.DisableChangeEvents {
		ic.gtmCtx.AddShardListener(ic.mongoConfig, gtmOpts, config.makeShardInsertHandler())
	}
}

func (ic *indexClient) clusterWait() {
	if ic.config.ClusterName != "" {
		if ic.enabled {
			infoLog.Printf("Starting work for cluster %s", ic.config.ClusterName)
		} else {
			infoLog.Printf("Pausing work for cluster %s", ic.config.ClusterName)
			ic.waitEnabled()
		}
	}
}

func (ic *indexClient) hasNewEvents() bool {
	if ic.lastTs.T > ic.lastTsSaved.T ||
		(ic.lastTs.T == ic.lastTsSaved.T && ic.lastTs.I > ic.lastTsSaved.I) {
		return true
	}
	return false
}

func (ic *indexClient) nextTokens() {
	if ic.hasNewEvents() {
		ic.bulk.Flush()
		if err := ic.saveTokens(); err == nil {
			ic.lastTsSaved = ic.lastTs
		} else {
			ic.processErr(err)
		}
	}
}

func (ic *indexClient) nextTimestamp() {
	if ic.hasNewEvents() {
		ic.bulk.Flush()
		if err := ic.saveTimestamp(); err == nil {
			ic.lastTsSaved = ic.lastTs
		} else {
			ic.processErr(err)
		}
	}
}

func (ic *indexClient) nextStats() {
	if ic.config.IndexStats {
		if err := ic.doIndexStats(); err != nil {
			errorLog.Printf("Error indexing statistics: %s", err)
		}
	} else {
		stats, err := json.Marshal(ic.bulk.Stats())
		if err != nil {
			errorLog.Printf("Unable to log statistics: %s", err)
		} else {
			statsLog.Println(string(stats))
		}
	}
}

func (ic *indexClient) waitEnabled() {
	var err error
	heartBeat := time.NewTicker(10 * time.Second)
	defer heartBeat.Stop()
	wait := true
	for wait {
		select {
		case req := <-ic.statusReqC:
			req.responseC <- nil
		case <-heartBeat.C:
			ic.enabled, err = ic.enableProcess()
			if err != nil {
				ic.processErr(err)
			}
			if ic.enabled {
				wait = false
				infoLog.Printf("Resuming work for cluster %s", ic.config.ClusterName)
			}
		}
	}
}

func (ic *indexClient) nextHeartbeat() {
	var err error
	if ic.enabled {
		ic.enabled, err = ic.ensureEnabled()
		if err != nil {
			ic.processErr(err)
		}
		if !ic.enabled {
			infoLog.Printf("Pausing work for cluster %s", ic.config.ClusterName)
			ic.pauseWork()
		}
	} else {
		ic.enabled, err = ic.enableProcess()
		if err != nil {
			ic.processErr(err)
		}
		if ic.enabled {
			infoLog.Printf("Resuming work for cluster %s", ic.config.ClusterName)
			ic.resumeWork()
		}
	}
}

func (ic *indexClient) eventLoop() {
	var err error
	var allOpsVisited bool
	timestampTicker := time.NewTicker(10 * time.Second)
	if ic.config.Resume == false {
		timestampTicker.Stop()
	}
	heartBeat := time.NewTicker(10 * time.Second)
	if ic.config.ClusterName == "" {
		heartBeat.Stop()
	}
	statsTimeout := time.Duration(30) * time.Second
	if ic.config.StatsDuration != "" {
		statsTimeout, _ = time.ParseDuration(ic.config.StatsDuration)
	}
	printStats := time.NewTicker(statsTimeout)
	if ic.config.Stats == false {
		printStats.Stop()
	}
	infoLog.Println("Listening for events")
	ic.sigH.clientStartedC <- ic
	for {
		select {
		case timeout := <-ic.doneC:
			ic.enabled = false
			ic.shutdown(timeout)
			return
		case <-timestampTicker.C:
			if !ic.enabled {
				break
			}
			if ic.config.ResumeStrategy == tokenResumeStrategy {
				ic.nextTokens()
			} else {
				ic.nextTimestamp()
			}
		case <-heartBeat.C:
			if ic.config.ClusterName == "" {
				break
			}
			ic.nextHeartbeat()
		case <-printStats.C:
			if !ic.enabled {
				break
			}
			ic.nextStats()
		case req := <-ic.statusReqC:
			enabled, lastTs := ic.enabled, ic.lastTs
			statusResp := &statusResponse{
				enabled: enabled,
				lastTs:  lastTs,
			}
			req.responseC <- statusResp
		case err = <-ic.gtmCtx.ErrC:
			if err == nil {
				break
			}
			ic.processErr(err)
		case op, open := <-ic.gtmCtx.OpC:
			if !ic.enabled {
				break
			}
			if op == nil {
				if !open && !allOpsVisited {
					allOpsVisited = true
					ic.opsConsumed <- true
				}
				break
			}
			if op.IsSourceOplog() {
				ic.lastTs = op.Timestamp
				if ic.config.ResumeStrategy == tokenResumeStrategy {
					ic.tokens[op.ResumeToken.StreamID] = op.ResumeToken.ResumeToken
				}
			}
			if err = ic.routeOp(op); err != nil {
				ic.processErr(err)
			}
		}
	}
}

func (ic *indexClient) startIndex() {
	for i := 0; i < 5; i++ {
		ic.indexWg.Add(1)
		go func() {
			defer ic.indexWg.Done()
			for op := range ic.indexC {
				if err := ic.doIndex(op); err != nil {
					ic.processErr(err)
				}
			}
		}()
	}
}

func (ic *indexClient) startRelate() {
	if len(ic.config.Relate) > 0 {
		for i := 0; i < ic.config.RelateThreads; i++ {
			ic.relateWg.Add(1)
			go func() {
				defer ic.relateWg.Done()
				for op := range ic.relateC {
					if err := ic.processRelated(op); err != nil {
						ic.processErr(err)
					}
				}
			}()
		}
	}
}

func (ic *indexClient) startCluster() {
	if ic.config.ClusterName != "" {
		var err error
		if err = ic.ensureClusterTTL(); err == nil {
			infoLog.Printf("Joined cluster %s", ic.config.ClusterName)
		} else {
			errorLog.Fatalf("Unable to enable cluster mode: %s", err)
		}
		ic.enabled, err = ic.enableProcess()
		if err != nil {
			errorLog.Fatalf("Unable to determine enabled cluster process: %s", err)
		}
	}
}

func (ic *indexClient) closeClient() {
	if ic.mongo != nil && ic.config.ClusterName != "" {
		ic.resetClusterState()
	}
	if ic.hsc != nil {
		ic.hsc.shutdown = true
		ic.hsc.httpServer.Shutdown(context.Background())
	}
	if ic.bulk != nil {
		ic.bulk.Close()
	}
	if ic.bulkStats != nil {
		ic.bulkStats.Close()
	}
	close(ic.closeC)
}

func (ic *indexClient) shutdown(timeout int) {
	infoLog.Println("Shutting down")
	go ic.closeClient()
	doneC := make(chan bool)
	go func() {
		closeT := time.NewTimer(time.Duration(timeout) * time.Second)
		defer closeT.Stop()
		done := false
		for !done {
			select {
			case <-ic.closeC:
				done = true
				close(doneC)
			case <-closeT.C:
				done = true
				close(doneC)
			}
		}
	}()
	<-doneC
	os.Exit(exitStatus)
}

func getBuildInfo(client *mongo.Client) (bi *buildInfo, err error) {
	db := client.Database("admin")
	result := db.RunCommand(context.Background(), bson.M{
		"buildInfo": 1,
	})
	if err = result.Err(); err == nil {
		bi = &buildInfo{}
		err = result.Decode(bi)
	}
	return
}

func (ic *indexClient) saveTimestampFromServerStatus() {
	var err error
	db := ic.mongo.Database("admin")
	result := db.RunCommand(context.Background(), bson.M{
		"serverStatus": 1,
	})
	if err = result.Err(); err == nil {
		doc := &bsonx.Doc{}
		if err = result.Decode(doc); err == nil {
			var elem bsonx.Val
			elem, err = doc.LookupErr("operationTime")
			if err != nil {
				ic.processErr(err)
				return
			}
			if elem.Type() != bson.TypeTimestamp {
				err = fmt.Errorf("incorrect type for 'operationTime'. got %v. want %v", elem.Type(), bson.TypeTimestamp)
				ic.processErr(err)
				return
			}
			ic.lastTs = elem.Interface().(primitive.Timestamp)
			if err = ic.saveTimestamp(); err != nil {
				ic.processErr(err)
			}
		} else {
			ic.processErr(err)
		}
	} else {
		ic.processErr(err)
	}
	return
}

func (ic *indexClient) saveTimestampFromReplStatus() {
	if rs, err := gtm.GetReplStatus(ic.mongo); err == nil {
		if ic.lastTs, err = rs.GetLastCommitted(); err == nil {
			if err = ic.saveTimestamp(); err != nil {
				ic.processErr(err)
			}
		} else {
			ic.processErr(err)
		}
	} else {
		ic.saveTimestampFromServerStatus()
	}
}

func mustConfig() *configOptions {
	config := &configOptions{
		GtmSettings: gtmDefaultSettings(),
		LogRotate:   logRotateDefaults(),
	}
	config.parseCommandLineFlags()
	if config.Version {
		fmt.Println(version)
		os.Exit(0)
	}
	config.build()
	if config.Print {
		config.dump()
		os.Exit(0)
	}
	config.setupLogging()
	config.validate()
	return config
}

func validateFeatures(config *configOptions, mongoInfo *buildInfo) {
	if len(mongoInfo.VersionArray) < 2 {
		return
	}
	const featErr1 = "Change streams are not supported by the server before version 3.6 (see enable-oplog)"
	const featErr2 = "Resuming streams using timestamps requires server version 4.0 or greater (see resume-strategy)"
	const featErr3 = "A token based resume strategy is only supported for server version 3.6 or greater"
	major, minor := mongoInfo.VersionArray[0], mongoInfo.VersionArray[1]
	streamsSupported := major > 3 || (major == 3 && minor >= 6)
	startAtOperationTimeSupported := major >= 4
	streamsConfigured := len(config.ChangeStreamNs) > 0
	if streamsConfigured && !streamsSupported {
		errorLog.Println(featErr1)
	}
	if config.ResumeStrategy == timestampResumeStrategy {
		if streamsConfigured && !startAtOperationTimeSupported {
			errorLog.Println(featErr2)
		}
	} else if config.ResumeStrategy == tokenResumeStrategy {
		if !streamsSupported {
			errorLog.Println(featErr3)
		}
	}
	if config.ResumeFromTimestamp > 0 {
		if streamsConfigured && !startAtOperationTimeSupported {
			errorLog.Println(featErr2)
		}
	}
}

func buildMongoClient(config *configOptions) *mongo.Client {
	mongoClient, err := config.dialMongo(config.MongoURL)
	if err != nil {
		errorLog.Fatalf("Unable to connect to MongoDB using URL %s: %s",
			cleanMongoURL(config.MongoURL), err)
	}
	infoLog.Printf("Started monstache version %s", version)
	infoLog.Printf("Go version %s", runtime.Version())
	infoLog.Printf("MongoDB go driver %s", mongoversion.Driver)
	infoLog.Printf("Elasticsearch go driver %s", elastic.Version)
	if mongoInfo, err := getBuildInfo(mongoClient); err == nil {
		infoLog.Printf("Successfully connected to MongoDB version %s", mongoInfo.Version)
		validateFeatures(config, mongoInfo)
	} else {
		infoLog.Println("Successfully connected to MongoDB")
	}
	return mongoClient
}

func buildElasticClient(config *configOptions) *elastic.Client {
	elasticClient, err := config.newElasticClient()
	if err != nil {
		errorLog.Fatalf("Unable to create Elasticsearch client: %s", err)
	}
	if config.ElasticVersion == "" {
		if err := config.testElasticsearchConn(elasticClient); err != nil {
			errorLog.Fatalf("Unable to validate connection to Elasticsearch: %s", err)
		}
	} else {
		if err := config.parseElasticsearchVersion(config.ElasticVersion); err != nil {
			errorLog.Fatalf("Elasticsearch version must conform to major.minor.fix: %s", err)
		}
	}
	return elasticClient
}

func main() {

	config := mustConfig()

	sh := &sigHandler{
		clientStartedC: make(chan *indexClient),
	}
	sh.start()

	mongoClient := buildMongoClient(config)
	loadBuiltinFunctions(mongoClient, config)

	elasticClient := buildElasticClient(config)

	ic := &indexClient{
		config:      config,
		mongo:       mongoClient,
		client:      elasticClient,
		fileWg:      &sync.WaitGroup{},
		indexWg:     &sync.WaitGroup{},
		processWg:   &sync.WaitGroup{},
		relateWg:    &sync.WaitGroup{},
		opsConsumed: make(chan bool),
		closeC:      make(chan bool),
		doneC:       make(chan int),
		enabled:     true,
		indexC:      make(chan *gtm.Op),
		processC:    make(chan *gtm.Op),
		fileC:       make(chan *gtm.Op),
		relateC:     make(chan *gtm.Op, config.RelateBuffer),
		statusReqC:  make(chan *statusRequest),
		sigH:        sh,
		tokens:      bson.M{},
	}

	ic.run()
}
