import http from '@/api'

const find = async ({ bk_obj_id: objId, params, config }) => {
  try {
    const [{ info }, { count }] = await Promise.all([
      http.post(`search/instances/object/${objId}`, params, config),
      http.post(`count/instances/object/${objId}`, params)
    ])
    return { count, info: info || [] }
  } catch (error) {
    console.error(error)
    return { count: 0, info: [] }
  }
}

const findOne = async ({ bk_obj_id: objId, bk_inst_id: instId, config }) => {
  try {
    const { info } = await http.post(`search/instances/object/${objId}`, {
      page: { start: 0, limit: 1 },
      fields: [],
      conditions: {
        condition: 'AND',
        rules: [{
          field: 'bk_inst_id',
          operator: 'equal',
          value: instId
        }]
      }
    }, config)
    const [instance] = info || [null]
    return instance
  } catch (error) {
    console.error(error)
    return null
  }
}

export default {
  find,
  findOne
}
