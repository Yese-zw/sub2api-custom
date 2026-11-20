import adminAccessKeys from './adminAccessKeys'
import mcp from './mcp'
import docsCenter from './docsCenter'
import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import channelMonitorV2 from './channelMonitorV2'
import batchImage from './batchImage'
import admin from './admin'
import misc from './misc'

export default {
  adminAccessKeys,
  mcp,
  docsCenter,
  ...landing,
  ...common,
  ...dashboard,
  ...channelMonitorV2,
  ...batchImage,
  admin,
  ...misc,
}
