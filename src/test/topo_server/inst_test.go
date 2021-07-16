package topo_server_test

import (
	"context"
	"encoding/json"
	"fmt"

	"configcenter/src/common/metadata"
	commonutil "configcenter/src/common/util"
	"configcenter/src/test"
	"configcenter/src/test/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("inst test", func() {
	var instId, instId1 int64
	var propertyID1, propertyID2, uniqueID uint64

	It("create object bk_classification_id = 'bk_network' and bk_obj_id='cc_test'", func() {
		test.ClearDatabase()
		input := metadata.Object{
			ObjCls:     "bk_network",
			ObjIcon:    "icon-cc-business",
			ObjectID:   "cc_test",
			ObjectName: "cc_test",
			OwnerID:    "0",
			Creator:    "admin",
		}
		rsp, err := objectClient.CreateObject(context.Background(), header, input)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
	})

	It("create object attribute bk_obj_id='cc_test' and bk_property_id='test_sglchar' and bk_property_name='test_sglchar'", func() {
		input := &metadata.ObjAttDes{
			Attribute: metadata.Attribute{
				ObjectID:     "cc_test",
				PropertyID:   "test_sglchar",
				PropertyName: "test_sglchar",
				IsEditable:   false,
				PropertyType: "singlechar",
				Option:       "a+b*",
				IsRequired:   true,
			},
		}
		rsp, err := apiServerClient.CreateObjectAtt(context.Background(), header, input)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
		propertyID1Float64, err := commonutil.GetFloat64ByInterface(rsp.Data.(map[string]interface{})["id"])
		Expect(err).NotTo(HaveOccurred())
		propertyID1 = uint64(propertyID1Float64)
	})

	It("create object attribute bk_obj_id='cc_test' and bk_property_id='test_unique' and bk_property_name='test_unique'", func() {
		input := &metadata.ObjAttDes{
			Attribute: metadata.Attribute{
				ObjectID:     "cc_test",
				PropertyID:   "test_unique",
				PropertyName: "test_unique",
				IsEditable:   true,
				PropertyType: "singlechar",
				IsRequired:   false,
			},
		}
		rsp, err := apiServerClient.CreateObjectAtt(context.Background(), header, input)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
		propertyID2Float64, err := commonutil.GetFloat64ByInterface(rsp.Data.(map[string]interface{})["id"])
		Expect(err).NotTo(HaveOccurred())
		propertyID2 = uint64(propertyID2Float64)
	})

	It("create object attribute bk_obj_id='cc_test' same bk_property_id", func() {
		input := &metadata.ObjAttDes{
			Attribute: metadata.Attribute{
				ObjectID:     "cc_test",
				PropertyID:   "test_unique",
				PropertyName: "test_unique1",
				IsEditable:   true,
				PropertyType: "singlechar",
				IsRequired:   false,
			},
		}
		rsp, err := apiServerClient.CreateObjectAtt(context.Background(), header, input)
		util.RegisterResponse(rsp)
		Expect(err).Should(BeNil())
		Expect(rsp.Result).To(Equal(false))
	})

	It("create object attribute bk_obj_id='cc_test' same bk_property_name", func() {
		input := &metadata.ObjAttDes{
			Attribute: metadata.Attribute{
				ObjectID:     "cc_test",
				PropertyID:   "test_unique1",
				PropertyName: "test_unique",
				IsEditable:   true,
				PropertyType: "singlechar",
				IsRequired:   false,
			},
		}
		rsp, err := apiServerClient.CreateObjectAtt(context.Background(), header, input)
		util.RegisterResponse(rsp)
		Expect(err).Should(BeNil())
		Expect(rsp.Result).To(Equal(false))
	})

	It("create object attribute invalid bk_obj_id", func() {
		input := &metadata.ObjAttDes{
			Attribute: metadata.Attribute{
				ObjectID:     "abcdefg",
				PropertyID:   "test",
				PropertyName: "test",
				IsEditable:   true,
				PropertyType: "singlechar",
				IsRequired:   false,
			},
		}
		rsp, err := apiServerClient.CreateObjectAtt(context.Background(), header, input)
		util.RegisterResponse(rsp)
		Expect(err).Should(BeNil())
		Expect(rsp.Result).To(Equal(false))
	})

	It("create inst bk_obj_id='cc_test'", func() {
		input := map[string]interface{}{
			"test_sglchar": "ab",
			"bk_inst_name": "wejeidjew",
			"test_unique":  "1234",
		}
		rsp, err := instClient.CreateInst(context.Background(), "cc_test", header, input)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
		Expect(rsp.Data["bk_inst_name"].(string)).To(Equal("wejeidjew"))
		Expect(rsp.Data["test_sglchar"].(string)).To(Equal("ab"))
		Expect(rsp.Data["bk_obj_id"].(string)).To(Equal("cc_test"))
		Expect(rsp.Data["test_unique"].(string)).To(Equal("1234"))
	})

	It("create inst nonexist attribute", func() {
		input := map[string]interface{}{
			"test_sglchar": "ab",
			"bk_inst_name": "wejeidjew123",
			"test_unique":  "1234567",
			"test_123":     "123456",
		}
		rsp, err := instClient.CreateInst(context.Background(), "cc_test", header, input)
		util.RegisterResponse(rsp)
		Expect(err).Should(BeNil())
		Expect(rsp.Result).To(Equal(true))
		Expect(rsp.Data.Exists("test_123")).To(Equal(false))
	})

	It("create inst missing required field", func() {
		input := map[string]interface{}{
			"bk_inst_name": "wejeidjew4",
		}
		rsp, err := instClient.CreateInst(context.Background(), "cc_test", header, input)
		util.RegisterResponse(rsp)
		Expect(err).Should(BeNil())
		Expect(rsp.Result).To(Equal(false))
	})

	It("create object attribute unique", func() {
		input := &metadata.CreateUniqueRequest{
			MustCheck: false,
			Keys: []metadata.UniqueKey{
				{
					Kind: "property",
					ID:   propertyID1,
				},
				{
					Kind: "property",
					ID:   propertyID2,
				},
			},
		}
		rsp, err := objectClient.CreateObjectUnique(context.Background(), "cc_test", header, input)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
		uniqueIDFloat64, err := commonutil.GetFloat64ByInterface(rsp.Data.(map[string]interface{})["id"])
		Expect(err).NotTo(HaveOccurred())
		uniqueID = uint64(uniqueIDFloat64)
	})

	It("create object attribute unique with duplicate inst", func() {
		input := &metadata.CreateUniqueRequest{
			MustCheck: false,
			Keys: []metadata.UniqueKey{
				{
					Kind: "property",
					ID:   propertyID1,
				},
			},
		}
		rsp, err := objectClient.CreateObjectUnique(context.Background(), "cc_test", header, input)
		util.RegisterResponse(rsp)
		Expect(err).Should(BeNil())
		Expect(rsp.Result).To(Equal(false))
	})

	It("search object attribute unique", func() {
		rsp, err := objectClient.SearchObjectUnique(context.Background(), "cc_test", header)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
		j, err := json.Marshal(rsp.Data)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(j).To(ContainSubstring(fmt.Sprintf("\"id\":%d", uniqueID)))
		Expect(j).To(ContainSubstring(fmt.Sprintf("\"key_id\":%d", propertyID1)))
		Expect(j).To(ContainSubstring(fmt.Sprintf("\"key_id\":%d", propertyID2)))
	})

	It("create inst duplicate unique attribute", func() {
		input := map[string]interface{}{
			"test_sglchar": "ab",
			"bk_inst_name": "wejeidjew10",
			"test_unique":  "1234",
		}
		rsp, err := instClient.CreateInst(context.Background(), "cc_test", header, input)
		util.RegisterResponse(rsp)
		Expect(err).Should(BeNil())
		Expect(rsp.Result).To(Equal(false))
	})

	It("update object attribute unique", func() {
		input := &metadata.UpdateUniqueRequest{
			Keys: []metadata.UniqueKey{
				{
					Kind: "property",
					ID:   propertyID2,
				},
			},
		}
		rsp, err := objectClient.UpdateObjectUnique(context.Background(), "cc_test", header, uniqueID, input)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
	})

	It("update object attribute unique duplicate inst", func() {
		input := &metadata.UpdateUniqueRequest{
			Keys: []metadata.UniqueKey{
				{
					Kind: "property",
					ID:   propertyID1,
				},
			},
		}
		rsp, err := objectClient.UpdateObjectUnique(context.Background(), "cc_test", header, uniqueID, input)
		util.RegisterResponse(rsp)
		Expect(err).Should(BeNil())
		Expect(rsp.Result).To(Equal(false))
	})

	It("search object attribute unique", func() {
		rsp, err := objectClient.SearchObjectUnique(context.Background(), "cc_test", header)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
		j, err := json.Marshal(rsp.Data)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(j).To(ContainSubstring(fmt.Sprintf("\"id\":%d", uniqueID)))
		Expect(j).To(ContainSubstring(fmt.Sprintf("\"key_id\":%d", propertyID2)))
	})

	It("delete object attribute unique", func() {
		rsp, err := objectClient.DeleteObjectUnique(context.Background(), "cc_test", header, uniqueID)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
	})

	It("search object attribute unique", func() {
		rsp, err := objectClient.SearchObjectUnique(context.Background(), "cc_test", header)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
		j, err := json.Marshal(rsp.Data)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(j).NotTo(ContainSubstring(fmt.Sprintf("\"id\":%d", uniqueID)))
		Expect(j).NotTo(ContainSubstring(fmt.Sprintf("\"key_id\":%d", propertyID1)))
		Expect(j).NotTo(ContainSubstring(fmt.Sprintf("\"key_id\":%d", propertyID2)))
	})

	It("create inst duplicate once unique attribute", func() {
		input := map[string]interface{}{
			"test_sglchar": "ab",
			"bk_inst_name": "wejeidjew10",
			"test_unique":  "1234",
		}
		rsp, err := instClient.CreateInst(context.Background(), "cc_test", header, input)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
	})

	It("create inst bk_obj_id='bk_switch'", func() {
		test.ClearDatabase()
		input := map[string]interface{}{
			"bk_asset_id":  "123",
			"bk_inst_name": "wejeidjew",
			"bk_sn":        "1234",
		}
		rsp, err := instClient.CreateInst(context.Background(), "bk_switch", header, input)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
		Expect(rsp.Data["bk_inst_name"].(string)).To(Equal("wejeidjew"))
		Expect(rsp.Data["bk_asset_id"].(string)).To(Equal("123"))
		Expect(rsp.Data["bk_obj_id"].(string)).To(Equal("bk_switch"))
		Expect(rsp.Data["bk_sn"].(string)).To(Equal("1234"))
		instId, err = commonutil.GetInt64ByInterface(rsp.Data["bk_inst_id"])
		Expect(err).NotTo(HaveOccurred())
	})

	It("create inst bk_obj_id='bk_switch'", func() {
		input := map[string]interface{}{
			"bk_asset_id":  "234",
			"bk_inst_name": "wejeidjew",
			"bk_sn":        "1234",
		}
		rsp, err := instClient.CreateInst(context.Background(), "bk_switch", header, input)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
		Expect(rsp.Data["bk_inst_name"].(string)).To(Equal("wejeidjew"))
		Expect(rsp.Data["bk_asset_id"].(string)).To(Equal("234"))
		Expect(rsp.Data["bk_obj_id"].(string)).To(Equal("bk_switch"))
		Expect(rsp.Data["bk_sn"].(string)).To(Equal("1234"))
		instId1, err = commonutil.GetInt64ByInterface(rsp.Data["bk_inst_id"])
		Expect(err).NotTo(HaveOccurred())
	})

	It("create inst invalid bk_obj_id", func() {
		input := map[string]interface{}{
			"bk_asset_id":  "3456",
			"bk_inst_name": "1234567",
			"bk_sn":        "1234",
		}
		rsp, err := instClient.CreateInst(context.Background(), "abcdefg", header, input)
		util.RegisterResponse(rsp)
		Expect(err).Should(BeNil())
		Expect(rsp.Result).To(Equal(false))
	})

	It("create inst bk_obj_id='bk_switch' duplicate bk_asset_id", func() {
		input := map[string]interface{}{
			"bk_asset_id":  "234",
			"bk_inst_name": "abcdefg",
		}
		rsp, err := instClient.CreateInst(context.Background(), "bk_switch", header, input)
		util.RegisterResponse(rsp)
		Expect(err).Should(BeNil())
		Expect(rsp.Result).To(Equal(false))
	})

	It("create inst bk_obj_id='bk_switch' missing bk_inst_name", func() {
		input := map[string]interface{}{
			"bk_asset_id": "456",
		}
		rsp, err := instClient.CreateInst(context.Background(), "bk_switch", header, input)
		util.RegisterResponse(rsp)
		Expect(err).Should(BeNil())
		Expect(rsp.Result).To(Equal(false))
	})

	It("create inst bk_obj_id='bk_switch' missing bk_asset_id", func() {
		input := map[string]interface{}{
			"bk_inst_name": "456",
		}
		rsp, err := instClient.CreateInst(context.Background(), "bk_switch", header, input)
		util.RegisterResponse(rsp)
		Expect(err).Should(BeNil())
		Expect(rsp.Result).To(Equal(false))
	})

	It("update inst", func() {
		input := map[string]interface{}{
			"bk_inst_name": "aaa",
		}
		rsp, err := instClient.UpdateInst(context.Background(), "bk_switch", instId, header, input)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
	})

	It("update inst invalid instId", func() {
		input := map[string]interface{}{
			"bk_inst_name": "aaa",
		}
		rsp, err := instClient.UpdateInst(context.Background(), "bk_switch", int64(1000), header, input)
		util.RegisterResponse(rsp)
		Expect(err).Should(BeNil())
		Expect(rsp.Result).To(Equal(false))
	})

	It("update inst with mismatch object", func() {
		input := map[string]interface{}{
			"bk_inst_name": "123",
		}
		rsp, err := instClient.UpdateInst(context.Background(), "cc_test", instId, header, input)
		util.RegisterResponse(rsp)
		Expect(err).Should(BeNil())
		Expect(rsp.Result).To(Equal(false))
	})

	It("delete inst", func() {
		rsp, err := instClient.DeleteInst(context.Background(), "bk_switch", instId1, header)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
	})

	It("delete inst with mismatch object", func() {
		rsp, err := instClient.DeleteInst(context.Background(), "cc_test", instId, header)
		util.RegisterResponse(rsp)
		Expect(err).Should(BeNil())
		Expect(rsp.Result).To(Equal(false))
	})

	It("search inst", func() {
		input := &metadata.SearchParams{
			Condition: map[string]interface{}{},
			Page: map[string]interface{}{
				"sort": "id",
			},
		}
		rsp, err := instClient.SelectInsts(context.Background(), "0", "bk_switch", header, input)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
		Expect(rsp.Data.Count).To(Equal(1))
		Expect(rsp.Data.Info[0]["bk_inst_name"].(string)).To(Equal("aaa"))
		Expect(rsp.Data.Info[0]["bk_asset_id"].(string)).To(Equal("123"))
		Expect(rsp.Data.Info[0]["bk_obj_id"].(string)).To(Equal("bk_switch"))
		Expect(rsp.Data.Info[0]["bk_sn"].(string)).To(Equal("1234"))
	})

	It("search inst by object", func() {
		input := &metadata.SearchParams{
			Condition: map[string]interface{}{},
			Page: map[string]interface{}{
				"sort": "id",
			},
		}
		rsp, err := instClient.InstSearch(context.Background(), "bk_switch", header, input)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
		Expect(rsp.Data.Count).To(Equal(1))
		Expect(rsp.Data.Info[0]["bk_inst_name"].(string)).To(Equal("aaa"))
		Expect(rsp.Data.Info[0]["bk_asset_id"].(string)).To(Equal("123"))
		Expect(rsp.Data.Info[0]["bk_obj_id"].(string)).To(Equal("bk_switch"))
		Expect(rsp.Data.Info[0]["bk_sn"].(string)).To(Equal("1234"))
	})

	It("search inst association detail", func() {
		input := &metadata.SearchParams{
			Condition: map[string]interface{}{},
			Page: map[string]interface{}{
				"sort": "id",
			},
		}
		rsp, err := instClient.SelectInst(context.Background(), "bk_switch", instId, header, input)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
		Expect(rsp.Data.Count).To(Equal(1))
		Expect(rsp.Data.Info[0]["bk_inst_name"].(string)).To(Equal("aaa"))
		Expect(rsp.Data.Info[0]["bk_asset_id"].(string)).To(Equal("123"))
		Expect(rsp.Data.Info[0]["bk_obj_id"].(string)).To(Equal("bk_switch"))
		Expect(rsp.Data.Info[0]["bk_sn"].(string)).To(Equal("1234"))
	})

	It("search inst by association", func() {
		input := &metadata.AssociationParams{
			Page: metadata.BasePage{
				Sort: "id",
			},
		}
		rsp, err := instClient.SelectInstsByAssociation(context.Background(), "bk_switch", header, input)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
		Expect(rsp.Data.Count).To(Equal(1))
		Expect(rsp.Data.Info[0]["bk_inst_name"].(string)).To(Equal("aaa"))
		Expect(rsp.Data.Info[0]["bk_asset_id"].(string)).To(Equal("123"))
		Expect(rsp.Data.Info[0]["bk_obj_id"].(string)).To(Equal("bk_switch"))
		Expect(rsp.Data.Info[0]["bk_sn"].(string)).To(Equal("1234"))
	})

	It("search inst topo", func() {
		input := &metadata.SearchParams{
			Condition: map[string]interface{}{},
			Page: map[string]interface{}{
				"sort": "id",
			},
		}
		rsp, err := instClient.SelectTopo(context.Background(), "bk_switch", instId, header, input)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
	})

	It("search inst association topo", func() {
		input := &metadata.SearchParams{
			Condition: map[string]interface{}{},
			Page: map[string]interface{}{
				"sort": "id",
			},
		}
		rsp, err := instClient.SelectAssociationTopo(context.Background(), "bk_switch", instId, header, input)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
		j, err := json.Marshal(rsp.Data[0].Curr)
		data := map[string]interface{}{}
		json.Unmarshal(j, &data)
		Expect(data["bk_obj_id"].(string)).To(Equal("bk_switch"))
		Expect(data["bk_inst_name"].(string)).To(Equal("aaa"))
		instIdRes, err := commonutil.GetInt64ByInterface(data["bk_inst_id"])
		Expect(err).NotTo(HaveOccurred())
		Expect(instIdRes).To(Equal(instId))
	})

	It("get toponode host and serviceinst count", func() {
		cond1 := metadata.Condition{ObjID: "set", InstID: instId}
		conds := make([]metadata.Condition, 0)
		conds = append(conds, cond1)
		input := &metadata.SearchBizTopoNodeHostAndServiceInstCountOption{
			Condition: conds,
		}
		rsp, err := instClient.GetTopoNodeHostAndServiceInstCount(context.Background(), header, instId, input)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.Result).To(Equal(true))
	})
})

var _ = Describe("audit test", func() {
	It("search audit dict", func() {
		rsp, err := instClient.SearchAuditDict(context.Background(), header)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
	})

	It("search audit list", func() {
		input := &metadata.AuditQueryInput{
			Condition: metadata.AuditQueryCondition{
				OperationTime: metadata.OperationTimeCondition{
					Start: "2018-07-20 00:00:00",
					End:   "2018-07-21 23:59:59",
				},
			},
			Page: metadata.BasePage{
				Sort:  "-op_time",
				Limit: 10,
				Start: 0,
			},
		}
		rsp, err := instClient.SearchAuditList(context.Background(), header, input)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
	})

	It("search audit detail", func() {
		id := []int64{1}
		input := &metadata.AuditDetailQueryInput{
			IDs: id,
		}
		rsp, err := instClient.SearchAuditDetail(context.Background(), header, input)
		util.RegisterResponse(rsp)
		Expect(err).NotTo(HaveOccurred())
	})
})
