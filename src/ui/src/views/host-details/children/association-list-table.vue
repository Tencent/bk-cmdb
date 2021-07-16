<template>
  <div class="table" v-show="instances.length" v-bkloading="{ isLoading: $loading(propertyRequest, instanceRequest) }">
    <div class="table-info clearfix" @click="expanded = !expanded">
      <div class="info-title fl">
        <i class="icon bk-icon icon-right-shape"
          :class="{ 'is-open': expanded }">
        </i>
        <span class="title-text">{{title}}</span>
        <span class="title-count">({{instances.length}})</span>
      </div>
      <div class="info-pagination fr" v-show="pagination.count" @click.stop>
        <span class="pagination-info">{{getPaginationInfo()}}</span>
        <span class="pagination-toggle">
          <i class="pagination-icon bk-icon icon-cc-arrow-down left"
            :class="{ disabled: pagination.current === 1 }"
            @click="togglePage(-1)">
          </i>
          <i class="pagination-icon bk-icon icon-cc-arrow-down right"
            :class="{ disabled: pagination.current === totalPage }"
            @click="togglePage(1)">
          </i>
        </span>
      </div>
    </div>
    <bk-table class="association-table"
      v-show="expanded"
      :data="list"
      :max-height="462"
      :row-style="{ cursor: 'pointer' }"
      @row-click="handleShowDetails">
      <bk-table-column v-for="(column, index) in header"
        :key="column.id"
        :prop="column.id"
        :label="column.name"
        :class-name="index === 0 ? 'is-highlight' : ''"
        show-overflow-tooltip>
        <template slot-scope="{ row }">{{row[column.id] | formatter(column.property)}}</template>
      </bk-table-column>
      <bk-table-column :label="$t('操作')">
        <template slot-scope="{ row }">
          <cmdb-auth :auth="HOST_AUTH.U_HOST">
            <bk-button slot-scope="{ disabled }"
              text
              theme="primary"
              :disabled="disabled"
              @click.stop="showTips($event, row)">
              {{$t('取消关联')}}
            </bk-button>
          </cmdb-auth>
        </template>
      </bk-table-column>
      <cmdb-table-empty slot="empty" :stuff="table.stuff" :auth="permissionAuth"></cmdb-table-empty>
    </bk-table>
    <div class="confirm-tips" ref="confirmTips" v-show="confirm.show">
      <p class="tips-content">{{$t('确认取消')}}</p>
      <div class="tips-option">
        <bk-button class="tips-button" theme="primary" @click.stop="cancelAssociation">{{$t('确认')}}</bk-button>
        <bk-button class="tips-button" theme="default" @click.stop="hideTips">{{$t('取消')}}</bk-button>
      </div>
    </div>
  </div>
</template>

<script>
  import { mapGetters } from 'vuex'
  import authMixin from '../mixin-auth'
  import instanceService from '@/service/instance/instance'
  export default {
    name: 'cmdb-host-association-list-table',
    mixins: [authMixin],
    props: {
      type: {
        type: String,
        required: true
      },
      id: {
        type: String,
        required: true
      },
      associationType: {
        type: Object,
        required: true
      }
    },
    data() {
      return {
        properties: [],
        list: [],
        pagination: {
          count: 0,
          current: 1,
          size: 10
        },
        table: {
          stuff: {
            type: 'default',
            payload: {
              emptyText: this.$t('bk.table.emptyText')
            }
          }
        },
        expanded: false,
        confirm: {
          instance: null,
          item: null,
          target: null,
          id: null,
          show: false
        }
      }
    },
    computed: {
      ...mapGetters('hostDetails', [
        'allInstances',
      ]),
      hostId() {
        return parseInt(this.$route.params.id, 10)
      },
      model() {
        return this.$store.getters['objectModelClassify/getModelById'](this.id)
      },
      permissionAuth() {
        if (this.model.bk_obj_id === 'biz') {
          return {
            type: this.$OPERATION.R_BUSINESS
          }
        }
        return null
      },
      title() {
        const desc = this.type === 'source' ? this.associationType.src_des : this.associationType.dest_des
        return `${desc}-${this.model.bk_obj_name}`
      },
      propertyRequest() {
        return `get_${this.id}_association_list_table_properties`
      },
      instanceRequest() {
        return `get_${this.id}_association_list_table_instances`
      },
      page() {
        return {
          limit: this.pagination.size,
          start: (this.pagination.current - 1) * this.pagination.size
        }
      },
      totalPage() {
        return Math.ceil(this.pagination.count / this.pagination.size)
      },
      isSource() {
        return this.type === 'source'
      },
      instances() {
        const objAsstId = this.isSource
          ? `host_${this.associationType.bk_asst_id}_${this.id}`
          : `${this.id}_${this.associationType.bk_asst_id}_host`
        return this.allInstances.filter(instance => instance.bk_obj_asst_id === objAsstId)
      },
      instanceIds() {
        return this.instances.map(instance => (this.isSource ? instance.bk_asst_inst_id : instance.bk_inst_id))
      },
      header() {
        const headerProperties = this.$tools.getDefaultHeaderProperties(this.properties)
        const header = headerProperties.map(property => ({
          id: property.bk_property_id,
          name: this.$tools.getHeaderPropertyName(property),
          property
        }))
        return header
      },
      expandAll() {
        return this.$store.state.hostDetails.expandAll
      }
    },
    watch: {
      instances(instances) {
        instances.length && this.expanded && this.getData()
      },
      expandAll(expanded) {
        this.expanded = expanded
      },
      expanded(expanded) {
        expanded && this.getData()
      }
    },
    methods: {
      getData() {
        this.getProperties()
        this.getInstances()
      },
      async getProperties() {
        try {
          this.properties = await this.$store.dispatch('objectModelProperty/searchObjectAttribute', {
            params: {
              bk_obj_id: this.id
            },
            config: {
              fromCache: true,
              requestId: this.propertyRequest
            }
          })
        } catch (e) {
          console.error(e)
          this.properties = []
        }
      },
      async getInstances() {
        let promise
        const config = {
          requestId: this.instanceRequest,
          cancelPrevious: true,
          globalError: false,
          globalPermission: false
        }
        try {
          if (!this.instanceIds.length) {
            // 业务查询会走权限中心进行数据拼接，导致为空时返回了有权限的数据
            // 此处为空后不走查询
            promise = Promise.resolve({
              info: [],
              count: 0
            })
          } else {
            switch (this.id) {
              case 'host':
                promise = this.getHostInstances(config)
                break
              case 'biz':
                promise = this.getBusinessInstances(config)
                break
              default:
                promise = this.getModelInstances(config)
            }
          }
          const data = await promise
          this.list = data.info
          this.pagination.count = data.count
          if (data.count && !data.info.length) {
            this.pagination.current -= 1
            this.getInstances()
          }
        } catch (e) {
          console.error(e)
          this.list = []
          this.pagination.count = 0
        }
      },
      getHostInstances(config) {
        const models = ['biz', 'set', 'module', 'host']
        const hostCondition = {
          field: 'bk_host_id',
          operator: '$in',
          value: this.instanceIds
        }
        const condition = models.map(model => ({
          bk_obj_id: model,
          fields: [],
          condition: model === 'host' ? [hostCondition] : []
        }))
        return this.$store.dispatch('hostSearch/searchHost', {
          params: {
            bk_biz_id: -1,
            condition,
            id: {
              data: [],
              exact: 0,
              flag: 'bk_host_innerip|bk_host_outerip'
            },
            page: {
              ...this.page,
              sort: 'bk_host_id'
            }
          },
          config
        }).then(data => ({
          count: data.count,
          info: data.info.map(item => item.host)
        }))
      },
      getBusinessInstances(config) {
        return this.$store.dispatch('objectBiz/searchBusiness', {
          params: {
            condition: {
              bk_biz_id: {
                $in: this.instanceIds
              }
            },
            fields: [],
            page: {
              ...this.page,
              sort: 'bk_biz_id'
            }
          },
          config
        })
      },
      getModelInstances(config) {
        return instanceService.find({
          bk_obj_id: this.id,
          params: {
            fields: [],
            page: {
              ...this.page,
              sort: 'bk_inst_id'
            },
            conditions: {
              condition: 'AND',
              rules: [{
                field: 'bk_inst_id',
                operator: 'in',
                value: this.instanceIds
              }]
            }
          },
          config
        })
      },
      async cancelAssociation() {
        try {
          const instance = this.instances.find((instance) => {
            const key = this.isSource ? 'bk_asst_inst_id' : 'bk_inst_id'
            return instance[key] === this.confirm.id
          })
          await this.$store.dispatch('objectAssociation/deleteInstAssociation', {
            id: instance.id,
            objId: 'host',
            config: { data: {} }
          })
          this.$store.commit('hostDetails/deleteAssociation', instance.id)
          this.$success(this.$t('取消关联成功'))
          this.hideTips()
        } catch (e) {
          console.error(e)
        }
      },
      getPaginationInfo() {
        return this.$tc('页码', this.pagination.current, {
          current: this.pagination.current,
          count: this.pagination.count,
          total: this.totalPage
        })
      },
      togglePage(step) {
        const { current } = this.pagination
        const newCurrent = current + step
        if (newCurrent < 1 || newCurrent > this.totalPage) {
          return false
        }
        this.pagination.current = newCurrent
        this.getInstances()
      },
      hideTips() {
        this.confirm.instance && this.confirm.instance.hide()
      },
      getRowInstId(item) {
        const specialModel = ['host', 'biz', 'set', 'module']
        const mapping = {}
        specialModel.forEach(key => (mapping[key] = `bk_${key}_id`))
        return item[mapping[this.id] || 'bk_inst_id']
      },
      showTips(event, item) {
        this.confirm.item = item
        this.confirm.id = this.getRowInstId(item)
        this.confirm.instance = this.$bkPopover(event.target, {
          content: this.$refs.confirmTips,
          theme: 'light',
          zIndex: 9999,
          width: 200,
          trigger: 'click',
          boundary: 'window',
          arrow: true,
          interactive: true,
          onHidden: () => {
            this.confirm.instance && this.confirm.instance.destroy()
            this.confirm.instance = null
          }
        })
        this.confirm.show = true
        this.$nextTick(() => {
          this.confirm.instance.show()
        })
      },
      async handleShowDetails(row) {
        const showInstanceDetails = await import('@/components/instance/details')
        const nameMapping = { host: 'bk_host_innerip', biz: 'bk_biz_name' }
        const idMapping = { host: 'bk_host_id', biz: 'bk_biz_id' }
        showInstanceDetails.default({
          bk_obj_id: this.id,
          bk_inst_id: row[idMapping[this.id] || 'bk_inst_id'],
          title: `${this.model.bk_obj_name}-${row[nameMapping[this.id] || 'bk_inst_name']}`
        })
      }
    }
  }
</script>

<style lang="scss" scoped>
    .table {
        margin: 0 0 12px 0;
    }
    .table-info {
        height: 42px;
        padding: 0 20px;
        border-radius: 2px 2px 0 0;
        line-height: 42px;
        background-color: #DCDEE5;
        font-size: 14px;
    }
    .info-title {
        cursor: pointer;
        .icon {
            display: inline-block;
            vertical-align: middle;
            transition: transform .2s linear;
            &.is-open {
                transform: rotate(90deg);
            }
        }
        .title-text {
            color: #000;
        }
        .title-count {
            color: #8b8d95;
        }
    }
    .info-pagination {
        color: #8b8d95;
        .pagination-toggle {
            margin-left: 10px;
            .pagination-icon {
                font-size: 14px;
                color: #979BA5;
                cursor: pointer;
                &.disabled {
                    color: #C4C6CC;
                    cursor: not-allowed;
                }
                &.left {
                    transform: rotate(90deg);
                }
                &.right {
                    transform: rotate(-90deg);
                }
            }
        }
    }
    .confirm-tips {
        padding: 9px 0;
        text-align: center;
        .tips-content {
            color: $cmdbTextColor;
            line-height: 20px;
        }
        .tips-option {
            margin: 12px 0 0 0;
            .tips-button {
                height: 26px;
                line-height: 24px;
                padding: 0 16px;
                min-width: 56px;
                font-size: 12px;
            }
        }
    }
</style>
