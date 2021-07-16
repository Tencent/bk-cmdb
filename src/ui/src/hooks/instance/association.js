import service from '@/service/instance/association'
import { ref, reactive, set } from '@vue/composition-api'
import hasOwnProperty from 'has'
export default function () {
  const instanceMap = reactive({})
  const pending = ref(true)
  const find = async (options) => {
    const response = await service.findTopology(options)
    if (hasOwnProperty(instanceMap, options.bk_obj_id)) {
      set(instanceMap[options.bk_obj_id], options.bk_inst_id, response)
    } else {
      set(instanceMap, options.bk_obj_id, { [options.bk_inst_id]: response })
    }
    pending.value = false
  }
  return [{ map: instanceMap, pending }, find]
}
