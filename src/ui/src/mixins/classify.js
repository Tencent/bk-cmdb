import { mapGetters } from 'vuex'
import has from 'has'

export default {
  computed: {
    ...mapGetters('objectModelClassify', {
      $authorizedNavigation: 'authorizedNavigation',
      $classifications: 'classifications'
    }),
    $classify() {
      let $classify = {}
      const relativePath = this.$route.meta.relative || this.$route.query.relative || null
      const path = relativePath || this.$route.path
      for (let i = 0; i < this.$authorizedNavigation.length; i++) {
        const classify = this.$authorizedNavigation[i]
        if (has(classify, 'path') && classify.path === path) {
          $classify = classify
          break
        }
        if (classify.children && classify.children.length) {
          const targetModel = classify.children.find(child => child.path === path || child.relative === path)
          if (targetModel) {
            $classify = targetModel
            break
          }
        }
      }
      return $classify
    },
    $allModels() {
      const allModels = []
      this.$classifications.forEach((classify) => {
        classify.bk_objects.forEach((model) => {
          allModels.push(model)
        })
      })
      return allModels
    },
    $model() {
      const objId = this.$route.params.objId || this.$route.meta.objId
      const targetModel = this.$allModels.find(model => model.bk_obj_id === objId)
      return targetModel || {}
    }
  }
}
