import overview from './overview'
import channels from './channels'
import accounts from './accounts'
import resources from './resources'
import ops from './ops'
import settings from './settings'
import audit from './audit'
import promptAudit from './promptAudit'
import financial from './financial'
import plugins from './plugins'
import schedulingCenter from './schedulingCenter'
import operationsCenter from './operationsCenter'

export default {
  ...overview,
  ...channels,
  ...accounts,
  ...resources,
  ...ops,
  ...settings,
  ...audit,
  ...promptAudit,
  ...financial,
  ...plugins,
  ...schedulingCenter,
  ...operationsCenter,
}
