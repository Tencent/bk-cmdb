import http from '@/api'
import i18n from '@/i18n'
const getIdKey = modelId => ({ host: 'bk_host_id', biz: 'bk_biz_id' }[modelId] || 'bk_inst_id')
const getNameKey = modelId => ({ host: 'bk_host_innerip', biz: 'bk_biz_name' }[modelId] || 'bk_inst_name')
const findInstance = (instances, objId, instId) => {
  const idKey = getIdKey(objId)
  return (instances || []).find(instance => instance[idKey] === instId)
}

const findTopology = async ({
  bk_obj_id: currentModelId,
  bk_inst_id: currentInstId,
  bk_inst_name: currentInstName,
  offset = 0,
  limit = 200
}) => {
  try {
    // eslint-disable-next-line max-len
    const url = `findmany/inst/association/object/${currentModelId}/inst_id/${currentInstId}/offset/${offset}/limit/${limit}/web`
    const result = await http.post(url, {})
    // 忽略实例作为源还是目标，抹平不同模型间的key差异
    const all =  [...(result.data.association.dst || []), ...(result.data.association.src || [])]
    const data = all.map((association) => {
      const {
        bk_obj_id: objId,
        bk_inst_id: instId,
        bk_asst_obj_id: asstObjId,
        bk_asst_inst_id: asstInstId
      } = association
      const isSource = objId === currentModelId && instId === currentInstId
      const instance = isSource
        ? findInstance(result.data.instance[asstObjId], asstObjId, asstInstId)
        : findInstance(result.data.instance[objId], objId, instId)
      const nameKey = isSource ? getNameKey(asstObjId) : getNameKey(objId)
      return {
        id: association.id,
        bk_obj_id: isSource ? asstObjId : objId,
        bk_inst_id: isSource ? asstInstId : instId,
        bk_inst_name: instance ? instance[nameKey] : `${i18n.t('已删除的实例')}(ID: ${isSource ? asstInstId : instId})`,
        bk_asst_id: association.bk_asst_id,
        bk_obj_asst_id: association.bk_obj_asst_id,
        deleted: !instance,
        target: isSource
      }
    })
    const rootIdKey = getIdKey(currentModelId)
    const rootNameKey = getNameKey(currentModelId)
    const rootInstance = (result.data.instance[currentModelId] || []).find(root => root[rootIdKey] === currentInstId)
    return {
      count: result.association_count,
      root: {
        bk_obj_id: currentModelId,
        bk_inst_id: currentInstId,
        bk_inst_name: rootInstance ? rootInstance[rootNameKey] : currentInstName
      },
      data
    }
  } catch (error) {
    console.error(error)
    return { count: 0, data: [], root: { bk_obj_id: currentModelId, bk_inst_id: currentInstId } }
  }
}

const find = async (params, config) => {
  try {
    const [{ info }, [{ count }]] = await Promise.all([
      http.post(`search/instance_associations/object/${params.bk_obj_id}`, params, config),
      http.post(`count/instance_associations/object/${params.bk_obj_id}`, params)
    ])
    return { count, info: info || [] }
  } catch (error) {
    console.error(error)
    return { count: 0, info: [] }
  }
}

const MAX_LIMIT = 500
const findAll = async (params) => {
  try {
    const { count } = await http.post(`count/instance_associations/object/${params.bk_obj_id}`, params)
    if (count === 0) {
      return []
    }
    const requestProxy = Array(Math.ceil(count / MAX_LIMIT)).fill(null)
    const all = await Promise.all(requestProxy.map((_, index) => {
      const page = { start: index * MAX_LIMIT, limit: MAX_LIMIT }
      return http.post(`search/instance_associations/object/${params.bk_obj_id}`, {
        ...params,
        page
      })
    }))
    return all.reduce((acc, { info }) => {
      acc.push(...info)
      return acc
    }, [])
  } catch (error) {
    return []
  }
}

export default {
  find,
  findAll,
  findTopology
}
