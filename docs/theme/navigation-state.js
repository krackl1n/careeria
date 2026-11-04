(() => {
  const sidebar = document.querySelector('.md-sidebar--primary');
  if (!sidebar) return;
  const tree = sidebar.querySelector('.md-nav--primary');
  const scrollwrap = sidebar.querySelector('.md-sidebar__scrollwrap');
  const toggles = () => tree.querySelectorAll('.md-nav__item--nested > input.md-nav__toggle');
  const storageKey = 'careeria.docs.navigation.v3';
  let pending;

  function absoluteLinks(root, base) {
    root.querySelectorAll('[href], [src]').forEach(element => {
      for (const attribute of ['href', 'src']) {
        const value = element.getAttribute(attribute);
        if (value && !value.startsWith('data:')) {
          element.setAttribute(attribute, new URL(value, base).href);
        }
      }
    });
  }
  absoluteLinks(document, location.href);

  function key(toggle) {
    const item = toggle.parentElement;
    const link = item.querySelector(':scope > .md-nav__container > a');
    if (link) return new URL(link.href).pathname;
    return [...item.querySelectorAll(':scope > label.md-nav__link')]
      .map(label => label.textContent.trim()).join('/');
  }

  function expand(toggle, checked) {
    toggle.checked = checked;
    toggle.parentElement.querySelector(':scope > nav')
      ?.setAttribute('aria-expanded', String(checked));
  }

  function save() {
    try {
      sessionStorage.setItem(storageKey, JSON.stringify({
        sections: Object.fromEntries([...toggles()].map(toggle => [key(toggle), toggle.checked])),
        scroll: scrollwrap.scrollTop,
      }));
    } catch { /* Storage is optional. The live tree never depends on it. */ }
  }

  // Restore only on a full browser load. Page navigation below keeps these
  // exact DOM nodes, checkboxes and scroll position; nothing is replayed.
  try {
    const state = JSON.parse(sessionStorage.getItem(storageKey));
    if (state?.sections) {
      toggles().forEach(toggle => {
        if (typeof state.sections[key(toggle)] === 'boolean') expand(toggle, state.sections[key(toggle)]);
      });
      scrollwrap.scrollTop = state.scroll || 0;
    }
  } catch { /* A fresh tab uses the server-rendered initial tree. */ }

  // Material remounts and auto-scrolls every sidebar on document$.next().
  // Own the primary sidebar's layout so content widgets can be reinitialized
  // without ever remounting or auto-scrolling this tree.
  sidebar.setAttribute('data-md-component', 'persistent-navigation');
  function layout() {
    const header = document.querySelector('.md-header');
    const height = header?.getBoundingClientRect().height || 0;
    sidebar.style.top = `${height}px`;
    scrollwrap.style.height = `calc(100dvh - ${height + 24}px)`;
  }
  window.addEventListener('resize', layout);
  layout();

  function highlight(url) {
    tree.querySelectorAll('.md-nav__link--active').forEach(link => {
      link.classList.remove('md-nav__link--active');
      link.removeAttribute('aria-current');
    });
    tree.querySelectorAll('.md-nav__item--active').forEach(item => item.classList.remove('md-nav__item--active'));
    tree.querySelectorAll('a.md-nav__link').forEach(link => {
      if (new URL(link.href).pathname !== url.pathname) return;
      link.classList.add('md-nav__link--active');
      link.setAttribute('aria-current', 'page');
      let item = link.closest('.md-nav__item');
      while (item) {
        item.classList.add('md-nav__item--active');
        item = item.parentElement.closest('.md-nav__item');
      }
    });
  }

  async function navigate(url, push) {
    pending?.abort();
    const request = new AbortController();
    pending = request;
    try {
      const response = await fetch(url, {signal: request.signal});
      if (!response.ok || !response.headers.get('content-type')?.includes('text/html')) {
        throw new Error('Not a documentation page');
      }
      const page = new DOMParser().parseFromString(await response.text(), 'text/html');
      const content = page.querySelector('.md-content');
      if (!content) throw new Error('Missing page content');
      if (request.signal.aborted) return;
      absoluteLinks(page, url.href);
      if (push) {
        history.replaceState({...history.state, docsScroll: window.scrollY}, '');
        history.pushState({}, '', url);
      }
      document.querySelector('.md-content').replaceWith(content);
      const oldToc = document.querySelector('.md-sidebar--secondary');
      const newToc = page.querySelector('.md-sidebar--secondary');
      if (oldToc && newToc) oldToc.replaceWith(newToc);
      else if (oldToc) oldToc.remove();
      else if (newToc) content.before(newToc);
      const topic = document.querySelector('[data-md-component="header-topic"]');
      const newTopic = page.querySelector('[data-md-component="header-topic"]');
      if (topic && newTopic) topic.replaceChildren(...newTopic.childNodes);
      document.title = page.title;
      for (const rel of ['canonical', 'prev', 'next']) {
        document.head.querySelector(`link[rel="${rel}"]`)?.remove();
        const link = page.head.querySelector(`link[rel="${rel}"]`);
        if (link) document.head.append(link);
      }
      highlight(url);
      const scroll = scrollwrap.scrollTop;
      window.document$?.next(document);
      layout();
      scrollwrap.scrollTop = scroll;
      if (url.hash) document.getElementById(decodeURIComponent(url.hash.slice(1)))?.scrollIntoView();
      else window.scrollTo(0, push ? 0 : history.state?.docsScroll || 0);
      save();
    } catch (error) {
      if (error.name !== 'AbortError') location.assign(url.href);
    }
  }

  document.addEventListener('change', event => {
    if (tree.contains(event.target) && event.target.matches('.md-nav__item--nested > input.md-nav__toggle')) {
      expand(event.target, event.target.checked);
      save();
    }
  });
  document.addEventListener('click', event => {
    if (event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) return;
    const link = event.target.closest('a[href]');
    if (!link || link.target || link.hasAttribute('download')) return;
    const url = new URL(link.href);
    if (url.origin !== location.origin) return;
    if (url.pathname === location.pathname && url.search === location.search) {
      const container = link.closest('.md-nav__container');
      if (container && tree.contains(container) && !url.hash) {
        const toggle = container.parentElement.querySelector(':scope > input.md-nav__toggle');
        if (toggle) expand(toggle, true);
        save();
        event.preventDefault();
      }
      return;
    }
    if (!url.pathname.endsWith('/') && /\.[^/]+$/.test(url.pathname) && !url.pathname.endsWith('.html')) return;
    event.preventDefault();
    event.stopImmediatePropagation();
    const container = link.closest('.md-nav__container');
    if (container && tree.contains(container)) {
      const toggle = container.parentElement.querySelector(':scope > input.md-nav__toggle');
      if (toggle) expand(toggle, true);
    }
    save();
    navigate(url, true);
  }, true);
  window.addEventListener('popstate', () => navigate(new URL(location.href), false));
  window.addEventListener('pagehide', save);
})();
