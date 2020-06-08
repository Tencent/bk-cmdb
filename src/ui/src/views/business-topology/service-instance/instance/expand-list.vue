<template>
    <bk-table v-if="!serviceInstance.pending"
        :data="list"
        :outer-border="false"
        :header-cell-style="{ backgroundColor: '#fff' }">
        <bk-table-column v-for="property in header"
            :key="property.bk_property_id"
            :label="property.bk_property_name"
            :prop="property.bk_property_id">
            <cmdb-property-value slot-scope="{ row }"
                :theme="property.bk_property_id === 'bk_func_name' ? 'primary' : 'default'"
                :value="row.property[property.bk_property_id] | formatter(property)"
                :show-unit="false"
                :show-title="true"
                :property="property">
            </cmdb-property-value>
        </bk-table-column>
        <bk-table-column :label="$t('操作')">
            <template slot-scope="{ row }">
                <cmdb-auth class="mr10" :auth="{ type: $OPERATION.U_SERVICE_INSTANCE, bk_biz_id: bizId }">
                    <bk-button slot-scope="{ disabled }"
                        theme="primary" text
                        :disabled="disabled"
                        @click="handleEdit(row)">
                        {{$t('编辑')}}
                    </bk-button>
                </cmdb-auth>
                <cmdb-auth :auth="{ type: $OPERATION.U_SERVICE_INSTANCE, bk_biz_id: bizId }" v-if="!row.relation.service_template_id">
                    <bk-button slot-scope="{ disabled }"
                        theme="primary" text
                        :disabled="disabled"
                        @click="handleDelete(row)">
                        {{$t('删除')}}
                    </bk-button>
                </cmdb-auth>
            </template>
        </bk-table-column>
    </bk-table>
</template>

<script>
    import { processPropertyRequestId } from '../common/symbol'
    import { processTableHeader } from '@/dictionary/table-header'
    import { mapGetters } from 'vuex'
    import Form from '../common/form.js'
    import Bus from '../common/bus'
    export default {
        props: {
            serviceInstance: Object
        },
        data () {
            return {
                properties: [],
                header: [],
                list: [],
                request: {
                    list: Symbol('getList'),
                    delete: Symbol('delete')
                }
            }
        },
        computed: {
            ...mapGetters(['supplierAccount']),
            ...mapGetters('objectBiz', ['bizId'])
        },
        created () {
            this.getProperties()
            this.getList()
            Bus.$on('refresh-process-list', this.handleRefresh)
        },
        beforeDestroy () {
            Bus.$off('refresh-process-list', this.handleRefresh)
        },
        methods: {
            async getProperties () {
                try {
                    this.properties = await this.$store.dispatch('objectModelProperty/searchObjectAttribute', {
                        params: {
                            bk_obj_id: 'process',
                            bk_supplier_account: this.supplierAccount
                        },
                        config: {
                            requestId: processPropertyRequestId,
                            fromCache: true
                        }
                    })
                    this.setHeader()
                } catch (error) {
                    console.error(error)
                }
            },
            setHeader () {
                const header = []
                processTableHeader.forEach(id => {
                    const property = this.properties.find(property => property.bk_property_id === id)
                    if (property) {
                        header.push(property)
                    }
                })
                this.header = header
            },
            handleRefresh (target) {
                if (target !== this.serviceInstance) {
                    return
                }
                this.getList()
            },
            async getList () {
                try {
                    this.list = await this.$store.dispatch('processInstance/getServiceInstanceProcesses', {
                        params: {
                            bk_biz_id: this.bizId,
                            service_instance_id: this.serviceInstance.id
                        },
                        config: {
                            requestId: this.request.list,
                            cancelPrevious: true,
                            cancelWhenRouteChange: true
                        }
                    })
                } catch (error) {
                    console.error(error)
                } finally {
                    this.$emit('update-list', this.list)
                }
            },
            handleEdit (row) {
                Form.show({
                    type: 'update',
                    title: this.$t('编辑进程'),
                    instance: row.property,
                    hostId: row.relation.bk_host_id,
                    serviceTemplateId: this.serviceInstance.service_template_id,
                    processTemplateId: row.relation.process_template_id,
                    submitHandler: this.editSubmitHandler
                })
            },
            async editSubmitHandler (values, changedValues, instance) {
                try {
                    await this.$store.dispatch('processInstance/updateServiceInstanceProcess', {
                        params: {
                            bk_biz_id: this.bizId,
                            processes: [{ ...instance, ...values }]
                        }
                    })
                    this.getList()
                } catch (error) {
                    console.error(error)
                }
            },
            async handleDelete (row) {
                try {
                    await this.$store.dispatch('processInstance/deleteServiceInstanceProcess', {
                        config: {
                            data: {
                                bk_biz_id: this.bizId,
                                process_instance_ids: [row.property.bk_process_id]
                            },
                            requestId: this.request.delete
                        }
                    })
                    if (this.list.length === 1) {
                        this.$emit('update-list', [])
                    } else {
                        this.getList()
                    }
                } catch (error) {
                    console.error(error)
                }
            }
        }
    }
</script>