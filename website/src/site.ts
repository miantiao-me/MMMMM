import type { Lang } from './i18n'
import type { Plugin } from './plugins'

export const repo = 'https://github.com/miantiao-me/MMMMM'

export type Asset = { name: string; size: number; browser_download_url: string }
export type Release = { tag_name: string; published_at: string; assets: Asset[] }

declare const __RELEASE__: Release | null
// The latest release, as it was when the site was built (see vite.config.ts).
export const release = __RELEASE__

declare const __PLUGINS__: Plugin[]
// The plugins on GitHub, as they were when the site was built.
export const builtPlugins = __PLUGINS__
export const mygo = 'https://mygo.egoist.dev'

export const site = {
  name: 'MMMMM',
  subtitle: 'MyGo/Muse/Miantiao/Mobile/Music',
  description: {
    en: 'A desktop music player written in Go with MyGo, no WebView. It plays your local files out of the box, with tags, covers and lyrics, and everything else comes from plugins.',
    zh: '用 Go 与 MyGo 写成的桌面音乐播放器，原生界面，没有 WebView。开箱即可播放本地音乐，读取标签、封面与歌词；其余的一切，交给插件。',
  } satisfies Record<Lang, string>,
}
