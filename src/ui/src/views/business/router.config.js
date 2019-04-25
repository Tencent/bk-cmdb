import Meta from '@/router/meta'
import { NAV_BASIC_RESOURCE } from '@/dictionary/menu'
import {
    C_BUSINESS,
    U_BUSINESS,
    R_BUSINESS,
    BUSINESS_ARCHIVE
} from '@/dictionary/auth'

export const OPERATION = {
    R_BUSINESS,
    C_BUSINESS,
    U_BUSINESS,
    BUSINESS_ARCHIVE
}

const businessPath = '/business'
const historyPath = '/history/biz'

export default [{
    name: 'business',
    path: businessPath,
    component: () => import('./index.vue'),
    meta: new Meta({
        menu: {
            id: 'business',
            i18n: 'Nav["业务"]',
            path: businessPath,
            parent: NAV_BASIC_RESOURCE
        },
        auth: {
            operation: Object.values(OPERATION)
        }
    })
}, {
    name: 'businessHistory',
    path: historyPath,
    component: () => import('./archived.vue'),
    meta: new Meta({
        auth: {
            view: BUSINESS_ARCHIVE,
            operation: [BUSINESS_ARCHIVE]
        }
    })
}]
