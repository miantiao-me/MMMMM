import { createSignal, onCleanup, onMount, type JSX } from 'solid-js'
import { GitHubMark, Icon } from './Icon'
import { ThemeSwitch } from '../theme'
import { mygo, repo, site } from '../site'
import { LangLink, LangMenu, useT } from '../i18n'

// langPath is where the language menu links to, for a page that isn't one
// of the site's own, like the 404.
export function Layout(props: { children: JSX.Element; langPath?: string }) {
  return (
    <>
      <Header langPath={props.langPath} />
      <main>{props.children}</main>
      <Footer />
    </>
  )
}

function Header(props: { langPath?: string }) {
  const t = useT()
  // Hairline and frosted backing only once the page has scrolled under it.
  const [scrolled, setScrolled] = createSignal(false)
  onMount(() => {
    const on = () => setScrolled(window.scrollY > 8)
    on()
    window.addEventListener('scroll', on, { passive: true })
    onCleanup(() => window.removeEventListener('scroll', on))
  })
  return (
    <header class="nav" classList={{ scrolled: scrolled() }}>
      <div class="nav-inner">
        <LangLink to="/" class="brand" aria-label={site.name}>
          <img src="/favicon.png" alt="" width={28} height={28} />
          <span>{site.name}</span>
        </LangLink>
        <div class="nav-end">
          <LangLink to="/plugins" class="nav-link" aria-label={t('Plugins', '插件')}>
            <Icon name="plugin" size={16} />
            <span>{t('Plugins', '插件')}</span>
          </LangLink>
          <LangMenu path={props.langPath} />
          <ThemeSwitch class="nav-theme" />
          <a class="icon-link" href={repo} aria-label={t('Source code on GitHub', 'GitHub 上的源代码')}>
            <GitHubMark size={19} />
          </a>
        </div>
      </div>
    </header>
  )
}

function Footer() {
  const t = useT()
  return (
    <footer class="footer">
      <div class="footer-inner">
        <span>
          {site.name} · {t('Based on ', '基于 ')}
          <a href="https://github.com/egoist/mygo-music">MyGo Music</a>
          {t(' · Distributed under ', '，以 ')}
          <a href={`${repo}/blob/main/LICENSE`}>AGPL-3.0</a>
          {t(' · Mobile UI and design from ', ' 发行 · 移动端界面与设计源自 ')}
          <a href="https://github.com/Lakr233/MuseAmp">MuseAmp</a>
          {' ('}<a href={`${repo}/blob/main/LICENSE-MuseAmp`}>MIT</a>{')'}
          {t(' · Made with ', ' · 用 ')}
          <a href={mygo}>MyGo</a>
          {t('', ' 写成')}
        </span>
        <a class="icon-link" href={repo} aria-label={t('Source code on GitHub', 'GitHub 上的源代码')}>
          <GitHubMark size={17} />
        </a>
      </div>
    </footer>
  )
}
