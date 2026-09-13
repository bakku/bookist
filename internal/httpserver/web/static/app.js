(() => {
  const toggle = document.querySelector('[data-navigation-toggle]');
  const navigation = document.querySelector('#mobile-navigation');
  if (!toggle || !navigation) return;

  toggle.addEventListener('click', () => {
    const expanded = toggle.getAttribute('aria-expanded') === 'true';
    toggle.setAttribute('aria-expanded', String(!expanded));
    toggle.classList.toggle('is-active', !expanded);
    navigation.classList.toggle('is-active', !expanded);
  });
})();
