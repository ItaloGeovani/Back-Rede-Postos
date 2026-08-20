(function () {
  'use strict';

  var REDE_LUCENA_DEFAULT = '3c3d7985-0dff-4397-aad2-11da0447188c';

  function qs(name) {
    try {
      return new URLSearchParams(window.location.search).get(name) || '';
    } catch (e) {
      return '';
    }
  }

  function detectOS() {
    var ua = navigator.userAgent || navigator.vendor || window.opera || '';
    if (/iPad|iPhone|iPod/.test(ua) && !window.MSStream) return 'ios';
    if (/android/i.test(ua)) return 'android';
    return 'other';
  }

  function setHint(os) {
    var el = document.getElementById('os-hint');
    if (!el) return;
    if (os === 'ios') {
      el.textContent = 'Detectamos iPhone/iPad — a App Store está em destaque.';
    } else if (os === 'android') {
      el.textContent = 'Detectamos Android — a Google Play está em destaque.';
    } else {
      el.textContent = 'Escolha a loja do seu celular.';
    }
  }

  function showToast(msg) {
    var t = document.getElementById('toast');
    if (!t) return;
    t.textContent = msg || 'Código copiado';
    t.classList.remove('is-hidden');
    window.clearTimeout(showToast._timer);
    showToast._timer = window.setTimeout(function () {
      t.classList.add('is-hidden');
    }, 1800);
  }

  function applyCodigo(codigo) {
    var display = document.getElementById('codigo-display');
    var hint = document.getElementById('codigo-hint');
    var inline = document.getElementById('codigo-inline');
    var card = document.getElementById('code-card');
    var btn = document.getElementById('btn-copiar');
    var clean = String(codigo || '')
      .trim()
      .toUpperCase()
      .replace(/\s+/g, '');

    if (!clean) {
      if (card) card.classList.add('is-empty');
      if (display) display.textContent = '——';
      if (inline) inline.textContent = '——';
      if (hint) {
        hint.textContent =
          'Abra o link completo compartilhado pelo amigo para ver o código, ou peça o código de indicação.';
      }
      if (btn) btn.disabled = true;
      return '';
    }

    if (card) card.classList.remove('is-empty');
    if (display) display.textContent = clean;
    if (inline) inline.textContent = clean;
    if (hint) {
      hint.textContent = 'Use este código no cadastro do app Lucena+.';
    }
    if (btn) {
      btn.disabled = false;
      btn.onclick = function () {
        if (navigator.clipboard && navigator.clipboard.writeText) {
          navigator.clipboard.writeText(clean).then(
            function () {
              showToast('Código copiado');
            },
            function () {
              fallbackCopy(clean);
            }
          );
        } else {
          fallbackCopy(clean);
        }
      };
    }
    return clean;
  }

  function fallbackCopy(text) {
    var ta = document.createElement('textarea');
    ta.value = text;
    ta.setAttribute('readonly', '');
    ta.style.position = 'fixed';
    ta.style.left = '-9999px';
    document.body.appendChild(ta);
    ta.select();
    try {
      document.execCommand('copy');
      showToast('Código copiado');
    } catch (e) {
      showToast('Não foi possível copiar');
    }
    document.body.removeChild(ta);
  }

  function applyStores(cfg, os) {
    var iosBtn = document.getElementById('store-ios');
    var androidBtn = document.getElementById('store-android');
    var empty = document.getElementById('stores-empty');
    var bar = document.getElementById('bar-download');
    var barLink = document.getElementById('bar-download-link');

    var iosUrl = (cfg && cfg.url_loja_ios) || '';
    var androidUrl = (cfg && cfg.url_loja_android) || '';
    var any = false;

    if (iosBtn) {
      if (iosUrl) {
        iosBtn.href = iosUrl;
        iosBtn.classList.remove('is-hidden');
        iosBtn.classList.toggle('is-primary', os === 'ios');
        any = true;
      } else {
        iosBtn.classList.add('is-hidden');
      }
    }

    if (androidBtn) {
      if (androidUrl) {
        androidBtn.href = androidUrl;
        androidBtn.classList.remove('is-hidden');
        androidBtn.classList.toggle('is-primary', os === 'android');
        any = true;
      } else {
        androidBtn.classList.add('is-hidden');
      }
    }

    if (empty) {
      empty.classList.toggle('is-hidden', any);
    }

    var preferred = os === 'ios' ? iosUrl : os === 'android' ? androidUrl : androidUrl || iosUrl;
    if (bar && barLink) {
      if (preferred) {
        barLink.href = preferred;
        bar.classList.remove('is-hidden');
      } else {
        bar.classList.add('is-hidden');
      }
    }
  }

  function loadLojas(idRede, os) {
    var url = '/v1/public/app-lojas?id_rede=' + encodeURIComponent(idRede);
    return fetch(url, { headers: { Accept: 'application/json' } })
      .then(function (res) {
        if (!res.ok) throw new Error('lojas ' + res.status);
        return res.json();
      })
      .then(function (data) {
        applyStores(data || {}, os);
      })
      .catch(function () {
        applyStores({}, os);
      });
  }

  document.addEventListener('DOMContentLoaded', function () {
    var codigo = qs('codigo') || qs('c') || '';
    var idRede = qs('id_rede') || REDE_LUCENA_DEFAULT;
    var os = detectOS();
    setHint(os);
    applyCodigo(codigo);
    loadLojas(idRede, os);
  });
})();
