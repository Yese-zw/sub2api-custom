import type { RouteLocationNormalizedLoaded } from 'vue-router'

const siteOrigin = 'https://lingsiai.vip'

function setMeta(attr: 'name' | 'property', key: string, content: string) {
  let element = document.head.querySelector(`meta[${attr}="${key}"]`)
  if (!(element instanceof HTMLMetaElement)) {
    element = document.createElement('meta')
    element.setAttribute(attr, key)
    document.head.appendChild(element)
  }
  element.setAttribute('content', content)
}

function setCanonical(href: string | null) {
  const existing = document.head.querySelector('link[rel="canonical"]')
  if (!href) {
    existing?.remove()
    return
  }
  const link = existing instanceof HTMLLinkElement ? existing : document.createElement('link')
  link.setAttribute('rel', 'canonical')
  link.setAttribute('href', href)
  if (!existing) document.head.appendChild(link)
}

export function applyDocumentSEO(route: Pick<RouteLocationNormalizedLoaded, 'path' | 'meta'>, title: string) {
  const indexable = route.meta.indexable === true
  const description = indexable && typeof route.meta.seoDescription === 'string'
    ? route.meta.seoDescription.trim()
    : ''
  const path = route.path === '/' ? '/home' : route.path
  const canonical = indexable ? `${siteOrigin}${path}` : null

  setMeta('name', 'description', description)
  setMeta('name', 'robots', indexable ? 'index, follow' : 'noindex, nofollow')
  setMeta('property', 'og:title', title)
  setMeta('property', 'og:description', description)
  setMeta('property', 'og:url', canonical ?? `${siteOrigin}/home`)
  setMeta('property', 'og:site_name', '灵思智域')
  setMeta('name', 'twitter:title', title)
  setMeta('name', 'twitter:description', description)
  setCanonical(canonical)
}
