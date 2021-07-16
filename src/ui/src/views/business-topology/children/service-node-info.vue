<template>
  <div class="node-info"
    v-bkloading="{
      isLoading: $loading([
        'getModelProperties',
        'getModelPropertyGroups',
        'getNodeInstance',
        'updateNodeInstance',
        'deleteNodeInstance',
        'removeServiceTemplate'
      ])
    }"
  >
    <cmdb-permission v-if="permission" class="permission-tips" :permission="permission"></cmdb-permission>
    <cmdb-details class="topology-details"
      v-else-if="type === 'details'"
      :class="{ pt10: !isSetNode && !isModuleNode }"
      :properties="properties"
      :property-groups="propertyGroups"
      :inst="instance"
      :show-options="modelId !== 'biz' && editable">
      <node-extra-info slot="prepend" :instance="instance"></node-extra-info>
      <template slot="details-options">
        <cmdb-auth :auth="{ type: $OPERATION.U_TOPO, relation: [business] }">
          <template slot-scope="{ disabled }">
            <bk-button class="button-edit"
              theme="primary"
              :disabled="disabled"
              @click="handleEdit">
              {{$t('编辑')}}
            </bk-button>
          </template>
        </cmdb-auth>
        <cmdb-auth :auth="{ type: $OPERATION.D_TOPO, relation: [business] }">
          <template slot-scope="{ disabled }">
            <span class="inline-block-middle" v-if="moduleFromSetTemplate"
              v-bk-tooltips="$t('由集群模板创建的模块无法删除')">
              <bk-button class="btn-delete" hover-theme="danger" disabled>
                {{$t('删除节点')}}
              </bk-button>
            </span>
            <bk-button class="btn-delete" v-else
              hover-theme="danger"
              :disabled="disabled"
              @click="handleDelete">
              {{$t('删除节点')}}
            </bk-button>
          </template>
        </cmdb-auth>
      </template>
    </cmdb-details>
    <template v-else-if="type === 'update'">
      <cmdb-form class="topology-form"
        ref="form"
        :properties="properties"
        :property-groups="propertyGroups"
        :disabled-properties="disabledProperties"
        :inst="instance"
        :type="type"
        :save-auth="{ type: $OPERATION.U_TOPO, relation: [business] }"
        :is-main-line="true"
        @on-submit="handleSubmit"
        @on-cancel="handleCancel">
        <form-service-category slot="prepend"
          v-if="!withTemplate && isModuleNode"
          :instance="instance"
          @change="handleServiceCategoryChange">
        </form-service-category>
      </cmdb-form>
    </template>
  </div>
</template>

<script>
  import has from 'has'
  import debounce from 'lodash.debounce'
  import NodeExtraInfo from './node-extra-info'
  import FormServiceCategory from './form-service-category'
  import instanceService from '@/service/instance/instance'
  export default {
    components: {
      NodeExtraInfo,
      FormServiceCategory
    },
    props: {
      active: Boolean
    },
    data() {
      return {
        type: 'details',
        properties: [],
        disabledProperties: [],
        propertyGroups: [],
        instance: {},
        first: '',
        second: '',
        hasChange: false,
        templateInfo: {
          serviceTemplateName: this.$t('无'),
          serviceCategory: '',
          setTemplateName: this.$t('无')
        },
        refresh: null,
        permission: null
      }
    },
    computed: {
      business() {
        return this.$store.getters['objectBiz/bizId']
      },
      propertyMap() {
        return this.$store.state.businessHost.propertyMap
      },
      propertyGroupMap() {
        return this.$store.state.businessHost.propertyGroupMap
      },
      selectedNode() {
        return this.$store.state.businessHost.selectedNode
      },
      isModuleNode() {
        return this.selectedNode && this.selectedNode.data.bk_obj_id === 'module'
      },
      isSetNode() {
        return this.selectedNode && this.selectedNode.data.bk_obj_id === 'set'
      },
      isBlueking() {
        let rootNode = this.selectedNode || {}
        if (rootNode.parent) {
          // eslint-disable-next-line prefer-destructuring
          rootNode = rootNode.parents[0]
        }
        return rootNode.name === '蓝鲸'
      },
      modelId() {
        if (this.selectedNode) {
          return this.selectedNode.data.bk_obj_id
        }
        return null
      },
      nodeNamePropertyId() {
        if (this.modelId) {
          const map = {
            biz: 'bk_biz_name',
            set: 'bk_set_name',
            module: 'bk_module_name'
          }
          return map[this.modelId] || 'bk_inst_name'
        }
        return null
      },
      nodePrimaryPropertyId() {
        if (this.modelId) {
          const map = {
            biz: 'bk_biz_id',
            set: 'bk_set_id',
            module: 'bk_module_id'
          }
          return map[this.modelId] || 'bk_inst_id'
        }
        return null
      },
      withTemplate() {
        return this.isModuleNode && !!this.instance.service_template_id
      },
      withSetTemplate() {
        return this.isSetNode && !!this.instance.set_template_id
      },
      moduleFromSetTemplate() {
        return this.isModuleNode && !!this.selectedNode.parent.data.set_template_id
      },
      editable() {
        const editable = this.$store.state.businessHost.blueKingEditable
        return this.isBlueking ? this.isBlueking && editable : true
      }
    },
    watch: {
      modelId: {
        immediate: true,
        handler(modelId) {
          if (modelId && this.active) {
            this.initProperties()
          }
        }
      },
      selectedNode: {
        immediate: true,
        handler(node) {
          if (node && this.active) {
            this.getNodeDetails()
          }
        }
      },
      active: {
        immediate: true,
        handler(active) {
          if (active) {
            this.refresh && this.refresh()
          }
        }
      }
    },
    created() {
      this.refresh = debounce(() => {
        this.initProperties()
        this.getNodeDetails()
      }, 10)
    },
    methods: {
      async initProperties() {
        this.type = 'details'
        try {
          const [
            properties,
            groups
          ] = await Promise.all([
            this.getProperties(),
            this.getPropertyGroups()
          ])
          this.properties = properties
          this.propertyGroups = groups
        } catch (e) {
          console.error(e)
          this.properties = []
          this.propertyGroups = []
        }
      },
      async getNodeDetails() {
        this.type = 'details'
        await this.getInstance()
        this.disabledProperties = this.selectedNode.data.bk_obj_id === 'module' && this.withTemplate ? ['bk_module_name'] : []
      },
      async getProperties() {
        let properties = []
        const { modelId } = this
        if (has(this.propertyMap, modelId)) {
          properties = this.propertyMap[modelId]
        } else {
          const action = 'objectModelProperty/searchObjectAttribute'
          properties = await this.$store.dispatch(action, {
            params: {
              bk_biz_id: this.business,
              bk_obj_id: modelId,
              bk_supplier_account: this.$store.getters.supplierAccount
            },
            config: {
              requestId: 'getModelProperties'
            }
          })
          this.$store.commit('businessHost/setProperties', {
            id: modelId,
            properties
          })
        }
        this.insertNodeIdProperty(properties)
        return Promise.resolve(properties)
      },
      async getPropertyGroups() {
        let groups = []
        const { modelId } = this
        if (has(this.propertyGroupMap, modelId)) {
          groups = this.propertyGroupMap[modelId]
        } else {
          const action = 'objectModelFieldGroup/searchGroup'
          groups = await this.$store.dispatch(action, {
            objId: modelId,
            params: { bk_biz_id: this.business },
            config: {
              requestId: 'getModelPropertyGroups'
            }
          })
          this.$store.commit('businessHost/setPropertyGroups', {
            id: modelId,
            groups
          })
        }
        return Promise.resolve(groups)
      },
      async getInstance() {
        try {
          const { modelId } = this
          const promiseMap = {
            biz: this.getBizInstance,
            set: this.getSetInstance,
            module: this.getModuleInstance
          }
          this.instance = await (promiseMap[modelId] || this.getCustomInstance)()
          this.$store.commit('businessHost/setSelectedNodeInstance', this.instance)
        } catch (e) {
          console.error(e)
          this.instance = {}
        }
      },
      async getBizInstance() {
        try {
          const data = await this.$store.dispatch('objectBiz/searchBusiness', {
            params: {
              page: { start: 0, limit: 1 },
              fields: [],
              condition: {
                bk_biz_id: { $eq: this.selectedNode.data.bk_inst_id }
              }
            },
            config: {
              requestId: 'getNodeInstance',
              cancelPrevious: true,
              globalPermission: false
            }
          })
          return data.info[0]
        } catch ({ permission }) {
          this.permission = permission
          return {}
        }
      },
      async getSetInstance() {
        const { info: [instance] } = await this.$store.dispatch('objectSet/searchSet', {
          bizId: this.business,
          params: {
            page: { start: 0, limit: 1 },
            fields: [],
            condition: {
              bk_set_id: this.selectedNode.data.bk_inst_id
            }
          },
          config: {
            requestId: 'getNodeInstance',
            cancelPrevious: true
          }
        })
        return instance
      },
      insertNodeIdProperty(properties) {
        if (properties.find(property => property.bk_property_id === this.nodePrimaryPropertyId)) {
          return
        }
        const nodeNameProperty = properties.find(property => property.bk_property_id === this.nodeNamePropertyId)
        properties.unshift({
          ...nodeNameProperty,
          bk_property_id: this.nodePrimaryPropertyId,
          bk_property_name: this.$t('ID'),
          editable: false
        })
      },
      async getModuleInstance() {
        const data = await this.$store.dispatch('objectModule/searchModule', {
          bizId: this.business,
          setId: this.selectedNode.parent.data.bk_inst_id,
          params: {
            page: { start: 0, limit: 1 },
            fields: [],
            condition: {
              bk_module_id: this.selectedNode.data.bk_inst_id,
              bk_supplier_account: this.$store.getters.supplierAccount
            }
          },
          config: {
            requestId: 'getNodeInstance',
            cancelPrevious: true
          }
        })
        // eslint-disable-next-line prefer-destructuring
        const instance = data.info[0]
        return instance
      },
      async getCustomInstance() {
        return instanceService.findOne({
          bk_obj_id: this.modelId,
          bk_inst_id: this.selectedNode.data.bk_inst_id
        })
      },
      handleEdit() {
        this.type = 'update'
      },
      handleServiceCategoryChange(id) {
        this.$set(this.$refs.form.values, 'service_category_id', id)
      },
      async handleSubmit(value) {
        if (!await this.$validator.validateAll()) return
        const promiseMap = {
          set: this.updateSetInstance,
          module: this.updateModuleInstance
        }
        const nameMap = {
          set: 'bk_set_name',
          module: 'bk_module_name'
        }
        try {
          await (promiseMap[this.modelId] || this.updateCustomInstance)({ ...value, bk_biz_id: this.business })
          this.selectedNode.data.bk_inst_name = value[nameMap[this.modelId] || 'bk_inst_name']
          this.instance = Object.assign({}, this.instance, value)
          this.type = 'details'
          this.$success(this.$t('修改成功'))
        } catch (e) {
          console.error(e)
        }
      },
      updateSetInstance(value) {
        return this.$store.dispatch('objectSet/updateSet', {
          bizId: this.business,
          setId: this.selectedNode.data.bk_inst_id,
          params: { ...value },
          config: {
            requestId: 'updateNodeInstance'
          }
        })
      },
      updateModuleInstance(value) {
        return this.$store.dispatch('objectModule/updateModule', {
          bizId: this.business,
          setId: this.selectedNode.parent.data.bk_inst_id,
          moduleId: this.selectedNode.data.bk_inst_id,
          params: {
            bk_supplier_account: this.$store.getters.supplierAccount,
            ...value
          },
          config: {
            requestId: 'updateNodeInstance'
          }
        })
      },
      updateCustomInstance(value) {
        return this.$store.dispatch('objectCommonInst/updateInst', {
          objId: this.modelId,
          instId: this.selectedNode.data.bk_inst_id,
          params: { ...value },
          config: {
            requestId: 'updateNodeInstance'
          }
        })
      },
      handleCancel() {
        this.type = 'details'
      },
      async handleDelete() {
        const count = await this.getSelectedNodeHostCount()
        if (count) {
          this.$error(this.$t('目标包含主机, 不允许删除'))
          return
        }
        this.$bkInfo({
          title: `${this.$t('确定删除')} ${this.selectedNode.name}?`,
          confirmFn: async () => {
            const promiseMap = {
              set: this.deleteSetInstance,
              module: this.deleteModuleInstance
            }
            try {
              await (promiseMap[this.modelId] || this.deleteCustomInstance)()
              const { tree } = this.selectedNode
              const parentId = this.selectedNode.parent.id
              const nodeId = this.selectedNode.id
              tree.setSelected(parentId, {
                emitEvent: true
              })
              tree.removeNode(nodeId)
              this.$success(this.$t('删除成功'))
            } catch (e) {
              console.error(e)
            }
          }
        })
      },
      deleteSetInstance() {
        return this.$store.dispatch('objectSet/deleteSet', {
          bizId: this.business,
          setId: this.selectedNode.data.bk_inst_id,
          config: {
            requestId: 'deleteNodeInstance',
            data: { bk_biz_id: this.business }
          }
        })
      },
      deleteModuleInstance() {
        return this.$store.dispatch('objectModule/deleteModule', {
          bizId: this.business,
          setId: this.selectedNode.parent.data.bk_inst_id,
          moduleId: this.selectedNode.data.bk_inst_id,
          config: {
            requestId: 'deleteNodeInstance',
            data: { bk_biz_id: this.business }
          }
        })
      },
      deleteCustomInstance() {
        return this.$store.dispatch('objectCommonInst/deleteInst', {
          objId: this.modelId,
          instId: this.selectedNode.data.bk_inst_id,
          config: {
            requestId: 'deleteNodeInstance',
            data: { bk_biz_id: this.business }
          }
        })
      },
      async getSelectedNodeHostCount() {
        const defaultModel = ['biz', 'set', 'module', 'host', 'object']
        const modelInstKey = {
          biz: 'bk_biz_id',
          set: 'bk_set_id',
          module: 'bk_module_id',
          host: 'bk_host_id',
          object: 'bk_inst_id'
        }
        const conditionParams = {
          condition: defaultModel.map(model => ({
            bk_obj_id: model,
            condition: [],
            fields: []
          }))
        }
        const { selectedNode } = this
        const selectedModel = defaultModel.includes(selectedNode.data.bk_obj_id) ? selectedNode.data.bk_obj_id : 'object'
        const selectedModelCondition = conditionParams.condition.find(model => model.bk_obj_id === selectedModel)
        selectedModelCondition.condition.push({
          field: modelInstKey[selectedModel],
          operator: '$eq',
          value: selectedNode.data.bk_inst_id
        })
        const data = await this.$store.dispatch('hostSearch/searchHost', {
          params: {
            ...conditionParams,
            bk_biz_id: this.business,
            ip: {
              flag: 'bk_host_innerip|bk_host_outer',
              exact: 0,
              data: []
            },
            page: {
              start: 0,
              limit: 1,
              sort: ''
            }
          },
          config: {
            requestId: 'searchHosts',
            cancelPrevious: true
          }
        })
        return data && data.count
      }
    }
  }
</script>

<style lang="scss" scoped>
    .permission-tips {
        position: absolute;
        top: 35%;
        left: 50%;
        transform: translate(-50%, -50%);
    }
    .node-info {
        height: 100%;
        margin: 0 -20px;
    }
    .topology-details.details-layout {
        padding: 0;
        /deep/ {
            .property-group {
                padding-left: 36px;
            }
            .property-list {
                padding-left: 20px;
            }
            .details-options {
                width: 100%;
                margin: 0;
                padding-left: 56px;
            }
        }
    }
    .topology-form {
        /deep/ {
            .form-groups {
                padding: 0 20px;
            }
            .property-list {
                margin-left: 36px;
            }
            .property-item {
                max-width: 554px !important;
            }
            .form-options {
                padding: 10px 0 0 36px;
            }
        }
    }
    .button-edit {
        min-width: 76px;
        margin-right: 4px;
    }
    .btn-delete{
        min-width: 76px;
    }
</style>
