package parser

type matchRule map[string]func() *parseStream

func (ps *parseStream) index2KeywordMap() matchRule {
	return matchRule{
		// cache related
		"cache": ps.cacheRelated,
		//admin related
		"admin": ps.adminRelated,
		//netCollector related
		"collector": ps.netCollectorRelated,
		//event related
		"event": ps.eventRelated,
		//cloud related
		"cloud": ps.cloudRelated,
		//host related
		"dynamicgroup": ps.hostRelated,
		"usercustom":   ps.hostRelated,
		"host":         ps.hostRelated,
		"identifier":   ps.hostRelated,
		"system":       ps.hostRelated,
		"hosts":        ps.hostRelated,
		//topology related
		"biz":       ps.topology,
		"object":    ps.topology,
		"objectatt": ps.topology,
		"topo":      ps.topology,
		"module":    ps.topology,
		"set":       ps.topology,
	}
}

func (ps *parseStream) index3KeywordMap() matchRule {
	return matchRule{
		//process related
		"proc":      ps.processRelated,
		"operation": ps.processRelated,
		//cloud related
		"resource": ps.cloudRelated,
		"cloud":    ps.cloudRelated,
		//host related
		"module_relation": ps.hostRelated,
		"host_apply_rule": ps.hostRelated,
		"host_apply_plan": ps.hostRelated,
		"hosts":           ps.hostRelated,
		//topology related
		"audit":      ps.topology,
		"audit_dict": ps.topology,
		"audit_list": ps.topology,
		"full_text":  ps.topology,
		"cloudarea":  ps.topology,
		"topo":       ps.topology,
		"module":     ps.topology,
		"set":        ps.topology,
		//topologylatest related
		"objectunique":             ps.topologyLatest,
		"associationtype":          ps.topologyLatest,
		"objectassociation":        ps.topologyLatest,
		"topoassociationtype":      ps.topologyLatest,
		"instassociation":          ps.topologyLatest,
		"inst":                     ps.topologyLatest,
		"instance_associations":    ps.topologyLatest,
		"instance":                 ps.topologyLatest,
		"insttopo":                 ps.topologyLatest,
		"instassttopo":             ps.topologyLatest,
		"object":                   ps.topologyLatest,
		"instances":                ps.topologyLatest,
		"objecttopo":               ps.topologyLatest,
		"objectclassification":     ps.topologyLatest,
		"classificationobject":     ps.topologyLatest,
		"objectattgroup":           ps.topologyLatest,
		"objectattr":               ps.topologyLatest,
		"topomodelmainline":        ps.topologyLatest,
		"topoinst":                 ps.topologyLatest,
		"topopath":                 ps.topologyLatest,
		"topoinst_with_statistics": ps.topologyLatest,
	}
}

func (ps *parseStream) matchAuthRoute() *parseStream {
	if len(ps.RequestCtx.Elements) <= 2 {
		return ps
	}

	index2Map := ps.index2KeywordMap()
	if fn, exist := index2Map[ps.RequestCtx.Elements[2]]; exist {
		fn()
	}

	if len(ps.Attribute.Resources) != 0 {
		return ps
	}

	if len(ps.RequestCtx.Elements) > 3 {
		index3Map := ps.index3KeywordMap()
		if fn, exist := index3Map[ps.RequestCtx.Elements[3]]; exist {
			fn()
		}

		if len(ps.Attribute.Resources) != 0 {
			return ps
		}
	}

	return ps.topologyLatest().
		topology().
		hostRelated().
		cacheRelated().
		adminRelated().
		processRelated().
		eventRelated().
		cloudRelated()
}
