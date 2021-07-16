<template>
  <div class="details-layout">
    <bk-tab class="details-tab"
      type="unborder-card"
      :active.sync="active">
      <bk-tab-panel name="property" :label="$t('属性')">
        <cmdb-property
          :properties="properties"
          :property-groups="propertyGroups"
          :inst="inst">
        </cmdb-property>
      </bk-tab-panel>
      <bk-tab-panel name="association" :label="$t('关联')">
        <cmdb-relation
          v-if="active === 'association'"
          :obj-id="objId"
          :inst="inst">
        </cmdb-relation>
      </bk-tab-panel>
      <bk-tab-panel name="history" :label="$t('变更记录')">
        <cmdb-audit-history v-if="active === 'history'"
          resource-type="model_instance"
          category="resource"
          :resource-id="instId">
        </cmdb-audit-history>
      </bk-tab-panel>
    </bk-tab>
  </div>
</template>

<script>
  import { mapGetters, mapActions } from 'vuex'
  import cmdbProperty from '@/components/model-instance/property'
  import cmdbAuditHistory from '@/components/model-instance/audit-history'
  import cmdbRelation from '@/components/model-instance/relation'
  import instanceService from '@/service/instance/instance'
  export default {
    components: {
      cmdbProperty,
      cmdbAuditHistory,
      cmdbRelation
    },
    data() {
      return {
        inst: {},
        properties: [],
        propertyGroups: [],
        active: this.$route.query.tab || 'property'
      }
    },
    computed: {
      ...mapGetters(['supplierAccount', 'userName']),
      ...mapGetters('objectModelClassify', ['models', 'getModelById']),
      instId() {
        return parseInt(this.$route.params.instId, 10)
      },
      objId() {
        return this.$route.params.objId
      },
      model() {
        return this.getModelById(this.objId) || {}
      }
    },
    watch: {
      instId() {
        this.getData()
      },
      objId() {
        this.getData()
      }
    },
    created() {
      this.getData()
    },
    methods: {
      ...mapActions('objectModelFieldGroup', ['searchGroup']),
      ...mapActions('objectModelProperty', ['searchObjectAttribute']),
      setBreadcrumbs(inst) {
        this.$store.commit('setTitle', `${this.model.bk_obj_name}【${inst.bk_inst_name}】`)
      },
      async getData() {
        try {
          const [inst, properties, propertyGroups] = await Promise.all([
            this.getInstInfo(),
            this.getProperties(),
            this.getPropertyGroups()
          ])

          this.inst = inst
          this.properties = properties
          this.propertyGroups = propertyGroups

          this.setBreadcrumbs(inst)
        } catch (e) {
          console.error(e)
        }
      },
      getInstInfo() {
        return instanceService.findOne({ bk_obj_id: this.objId, bk_inst_id: this.instId })
      },
      async getProperties() {
        try {
          const properties = await this.searchObjectAttribute({
            params: {
              bk_obj_id: this.objId,
              bk_supplier_account: this.supplierAccount
            },
            config: {
              requestId: `post_searchObjectAttribute_${this.objId}`,
              fromCache: false
            }
          })

          return properties
        } catch (e) {
          console.error(e)
        }
      },
      async getPropertyGroups() {
        try {
          const propertyGroups = this.searchGroup({
            objId: this.objId,
            params: {},
            config: {
              fromCache: false,
              requestId: `post_searchGroup_${this.objId}`
            }
          })

          return propertyGroups
        } catch (e) {
          console.error(e)
        }
      }
    }
  }
</script>

<style lang="scss" scoped>
    .details-layout {
        overflow: hidden;
        .details-tab {
            min-height: 400px;
            /deep/ {
                .bk-tab-header {
                    padding: 0;
                    margin: 0 20px;
                }
                .bk-tab-section {
                    @include scrollbar-y;
                    padding-bottom: 10px;
                }
            }
        }
    }
</style>
