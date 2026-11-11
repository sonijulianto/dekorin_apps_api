(function () {
  'use strict';

  var seedEl = document.getElementById('dekorin-visual-editor-seed');
  var state = {};
  try {
    if (seedEl && seedEl.textContent) {
      state = JSON.parse(seedEl.textContent) || {};
    }
  } catch (e) {
    state = {};
  }

  window.__DEKORIN_EDIT_STATE__ = state;
  window.__DEKORIN_VISUAL_EDITOR_READY__ = true;

  var isEditEnabled = true;

  function notifyHost(payload) {
    var msg = Object.assign({ source: 'dekorin-visual-editor' }, payload || {});
    // 1. Flutter WebView JavaScriptChannel
    try {
      if (window.DekorinEditorChannel && typeof window.DekorinEditorChannel.postMessage === 'function') {
        window.DekorinEditorChannel.postMessage(JSON.stringify(msg));
      }
    } catch (e) {}
    // 2. Web Portal iframe parent
    try {
      if (window.parent && window.parent !== window) {
        window.parent.postMessage(msg, '*');
      }
    } catch (e) {}
  }

  // Inject non-invasive visual editor CSS
  var style = document.createElement('style');
  style.id = 'dekorin-visual-editor-style';
  style.textContent = [
    'body.dekorin-ve-active [data-edit-field] {',
    '  cursor: pointer !important;',
    '  position: relative;',
    '  transition: outline 0.15s ease, box-shadow 0.15s ease, transform 0.12s ease;',
    '}',
    'body.dekorin-ve-active [data-edit-field]:not([data-edit-type="bg_photo"]) {',
    '  outline: 1.5px dashed rgba(212, 165, 116, 0.85) !important;',
    '  outline-offset: 3px !important;',
    '  border-radius: 6px;',
    '}',
    'body.dekorin-ve-active img[data-edit-field] {',
    '  outline: 2.5px dashed #D4A574 !important;',
    '  outline-offset: 2px !important;',
    '}',
    'body.dekorin-ve-active [data-edit-field]:not([data-edit-type="bg_photo"]):active {',
    '  transform: scale(0.985);',
    '}',
    'body.dekorin-ve-active [data-edit-type="bg_photo"] {',
    '  outline: 2px dashed rgba(212, 165, 116, 0.7) !important;',
    '  outline-offset: -4px !important;',
    '}',
    '.dekorin-ve-bg-pill {',
    '  position: absolute !important;',
    '  top: 12px;',
    '  right: 12px;',
    '  z-index: 95 !important;',
    '  display: inline-flex !important;',
    '  align-items: center;',
    '  gap: 5px;',
    '  padding: 6px 11px;',
    '  border-radius: 999px;',
    '  background: rgba(44, 44, 44, 0.84) !important;',
    '  color: #FFFFFF !important;',
    '  border: 1px solid rgba(212, 165, 116, 0.85) !important;',
    '  font-family: "Plus Jakarta Sans", Arial, sans-serif !important;',
    '  font-size: 11px !important;',
    '  font-weight: 700 !important;',
    '  line-height: 1.2 !important;',
    '  letter-spacing: 0.2px;',
    '  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.28);',
    '  cursor: pointer !important;',
    '  backdrop-filter: blur(4px);',
    '  pointer-events: auto !important;',
    '  text-decoration: none !important;',
    '  text-transform: none !important;',
    '}',
    '.dekorin-ve-bg-pill.pos-bottom {',
    '  top: auto;',
    '  bottom: 12px;',
    '}',
    'body:not(.dekorin-ve-active) .dekorin-ve-bg-pill,',
    'body:not(.dekorin-ve-active) .dekorin-ve-empty-card {',
    '  display: none !important;',
    '}',
    '.dekorin-ve-empty-card {',
    '  margin: 14px auto;',
    '  max-width: 420px;',
    '  padding: 14px 16px;',
    '  border-radius: 16px;',
    '  border: 2px dashed #D4A574;',
    '  background: rgba(249, 243, 236, 0.94);',
    '  color: #2C2C2C;',
    '  font-family: "Plus Jakarta Sans", Arial, sans-serif;',
    '  text-align: center;',
    '  cursor: pointer;',
    '  box-shadow: 0 4px 14px rgba(0,0,0,0.06);',
    '  position: relative;',
    '  z-index: 20;',
    '}',
    '.dekorin-ve-empty-card-title {',
    '  font-size: 13px;',
    '  font-weight: 700;',
    '  color: #8B5E34;',
    '  margin-bottom: 4px;',
    '}',
    '.dekorin-ve-empty-card-sub {',
    '  font-size: 11px;',
    '  color: #64748B;',
    '}',
    'body.dekorin-ve-active .dekorin-ve-photo-wrap {',
    '  position: relative;',
    '  display: inline-block;',
    '}',
    'body.dekorin-ve-active .ta17-anim,',
    'body.dekorin-ve-active .ta3-anim,',
    'body.dekorin-ve-active .pn-anim,',
    'body.dekorin-ve-active .reveal-up {',
    '  opacity: 1 !important;',
    '  transform: none !important;',
    '}'
  ].join('\n');
  document.head.appendChild(style);

  function setEditMode(enabled) {
    isEditEnabled = !!enabled;
    if (isEditEnabled) {
      document.body.classList.add('dekorin-ve-active');
    } else {
      document.body.classList.remove('dekorin-ve-active');
    }
    notifyHost({ action: 'edit_mode_changed', enabled: isEditEnabled });
  }

  function ensureBgPhotoBadges() {
    var bgNodes = document.querySelectorAll('[data-edit-type="bg_photo"]');
    bgNodes.forEach(function (el) {
      var computed = window.getComputedStyle(el);
      if (computed.position === 'static') {
        el.style.position = 'relative';
      }
      if (!el.querySelector(':scope > .dekorin-ve-bg-pill')) {
        var pill = document.createElement('button');
        pill.type = 'button';
        pill.className = 'dekorin-ve-bg-pill';
        if (el.id === 'gate-cover' || el.id === 'cover') {
          pill.style.top = '18px';
          pill.style.right = '16px';
        }
        var label = el.getAttribute('data-edit-label') || 'Ganti Foto';
        pill.textContent = '📷 ' + label;
        pill.addEventListener('click', function (ev) {
          if (!isEditEnabled) return;
          ev.preventDefault();
          ev.stopPropagation();
          triggerEditElement(el);
        });
        el.appendChild(pill);
      }
    });
  }

  function attachEmptyPlaceholders() {
    // Only add helper cards for sections that were omitted by Go template because empty
    var container =
      document.querySelector('.pn4-right-col') ||
      document.querySelector('.jawa-app-container') ||
      document.body;

    var anchorBefore =
      document.getElementById('WISHES') ||
      document.getElementById('ucapan') ||
      document.querySelector('.ta17-closing-sec') ||
      document.querySelector('.ta3-closing-sec') ||
      document.querySelector('footer');

    function addEmptyCardIfMissing(id, selectorCheck, field, type, label, subtext) {
      if (document.getElementById(id)) return;
      if (selectorCheck && document.querySelector(selectorCheck)) return;
      if (!anchorBefore || !anchorBefore.parentNode) return;

      var card = document.createElement('div');
      card.id = id;
      card.className = 'dekorin-ve-empty-card';
      card.setAttribute('data-edit-field', field);
      card.setAttribute('data-edit-type', type);
      card.setAttribute('data-edit-label', label);
      card.innerHTML =
        '<div class="dekorin-ve-empty-card-title">+ ' + label + '</div>' +
        '<div class="dekorin-ve-empty-card-sub">' + subtext + '</div>';
      anchorBefore.parentNode.insertBefore(card, anchorBefore);
    }

    var isKhitan = (state.event_type || '').toLowerCase() === 'khitan';
    if (!isKhitan) {
      addEmptyCardIfMissing(
        'dekorin-ve-empty-lovestory',
        '#cerita',
        'love_story',
        'love_story',
        'Tambah Kisah Cinta (Love Story)',
        'Bagian ini belum diisi. Klik untuk menambah momen kisah cinta.'
      );
    }

    addEmptyCardIfMissing(
      'dekorin-ve-empty-gallery',
      '#GALLERY, #galeri',
      'gallery_images',
      'gallery',
      'Tambah Galeri Foto',
      'Belum ada foto galeri. Klik untuk mengunggah foto momen Anda.'
    );

    addEmptyCardIfMissing(
      'dekorin-ve-empty-gift',
      '#gift',
      'bank_1',
      'bank_1',
      'Tambah Amplop Digital / Rekening Hadiah',
      'Klik untuk mengisi nomor rekening bank atau alamat kirim kado.'
    );

    addEmptyCardIfMissing(
      'dekorin-ve-empty-turut',
      '[data-edit-field="turut_mengundang"]',
      'turut_mengundang',
      'textarea',
      'Tambah Daftar Turut Mengundang',
      'Klik untuk menambahkan nama keluarga besar yang turut mengundang.'
    );
  }

  function refreshEmptyPlaceholderSummaries() {
    var lsCard = document.getElementById('dekorin-ve-empty-lovestory');
    if (lsCard) {
      var hasLs = !!(state.love_story && String(state.love_story).trim());
      lsCard.querySelector('.dekorin-ve-empty-card-title').textContent = hasLs
        ? '✓ Kisah Cinta Diisi (Klik untuk Ubah)'
        : '+ Tambah Kisah Cinta (Love Story)';
      lsCard.querySelector('.dekorin-ve-empty-card-sub').textContent = hasLs
        ? 'Akan tampil penuh di undangan setelah disimpan.'
        : 'Bagian ini belum diisi. Klik untuk menambah momen kisah cinta.';
    }

    var galCard = document.getElementById('dekorin-ve-empty-gallery');
    if (galCard) {
      var urls = splitGallery(state.gallery_images);
      galCard.querySelector('.dekorin-ve-empty-card-title').textContent = urls.length > 0
        ? '✓ ' + urls.length + ' Foto Galeri Dipilih (Klik untuk Kelola)'
        : '+ Tambah Galeri Foto';
      galCard.querySelector('.dekorin-ve-empty-card-sub').textContent = urls.length > 0
        ? 'Akan tampil di galeri undangan setelah disimpan.'
        : 'Belum ada foto galeri. Klik untuk mengunggah foto momen Anda.';
    }

    var giftCard = document.getElementById('dekorin-ve-empty-gift');
    if (giftCard) {
      var hasBank = !!(state.bank_account_1 || state.bank_account_2 || state.gift_address);
      giftCard.querySelector('.dekorin-ve-empty-card-title').textContent = hasBank
        ? '✓ Amplop Digital Diisi (Klik untuk Ubah)'
        : '+ Tambah Amplop Digital / Rekening Hadiah';
      giftCard.querySelector('.dekorin-ve-empty-card-sub').textContent = hasBank
        ? [state.bank_name_1, state.bank_account_1, state.bank_holder_1].filter(Boolean).join(' • ')
        : 'Klik untuk mengisi nomor rekening bank atau alamat kirim kado.';
    }

    var turutCard = document.getElementById('dekorin-ve-empty-turut');
    if (turutCard) {
      var hasTurut = !!(state.turut_mengundang && String(state.turut_mengundang).trim());
      turutCard.querySelector('.dekorin-ve-empty-card-title').textContent = hasTurut
        ? '✓ Turut Mengundang Diisi (Klik untuk Ubah)'
        : '+ Tambah Daftar Turut Mengundang';
    }
  }

  function splitGallery(raw) {
    if (!raw) return [];
    return String(raw)
      .split(',')
      .map(function (s) { return s.trim(); })
      .filter(Boolean);
  }

  function triggerEditElement(el) {
    var field = el.getAttribute('data-edit-field') || '';
    var type = el.getAttribute('data-edit-type') || 'text';
    var label = el.getAttribute('data-edit-label') || field;
    var aspect = el.getAttribute('data-edit-aspect') || 'free';

    if (!field) return;

    if (type === 'photo' || type === 'bg_photo') {
      notifyHost({
        action: 'edit_photo',
        field: field,
        type: type,
        label: label,
        aspect: aspect,
        currentValue: state[field] || ''
      });
      return;
    }

    if (type === 'gallery') {
      notifyHost({
        action: 'edit_gallery',
        field: 'gallery_images',
        label: label,
        currentValue: state.gallery_images || ''
      });
      return;
    }

    if (type === 'love_story') {
      notifyHost({
        action: 'edit_love_story',
        field: 'love_story',
        label: label,
        currentValue: state.love_story || ''
      });
      return;
    }

    if (type === 'couple_nick') {
      notifyHost({
        action: 'edit_couple_nick',
        field: 'couple_nick',
        label: label,
        data: {
          event_type: state.event_type || 'wedding',
          groom_nickname: state.groom_nickname || '',
          bride_nickname: state.bride_nickname || '',
          groom_name: state.groom_name || '',
          bride_name: state.bride_name || '',
          child_nickname: state.child_nickname || '',
          child_name: state.child_name || '',
          title: state.title || ''
        }
      });
      return;
    }

    if (type === 'bank_1' || type === 'bank_2') {
      var idx = type === 'bank_2' ? '2' : '1';
      notifyHost({
        action: 'edit_bank',
        bankIndex: idx,
        label: label,
        data: {
          bank_name: state['bank_name_' + idx] || '',
          bank_account: state['bank_account_' + idx] || '',
          bank_holder: state['bank_holder_' + idx] || ''
        }
      });
      return;
    }

    notifyHost({
      action: 'edit_text',
      field: field,
      type: type,
      label: label,
      currentValue: state[field] !== undefined && state[field] !== null
        ? String(state[field])
        : (el.textContent || '').trim()
    });
  }

  // Capture clicks globally in Edit Mode so anchors/buttons inside editable nodes don't navigate away
  document.addEventListener(
    'click',
    function (e) {
      if (!isEditEnabled) return;
      var target = e.target;
      if (!target || typeof target.closest !== 'function') return;

      // Allow the "Buka Undangan" gate button to work normally!
      if (target.closest('#ta17OpenBtn, #ta3OpenBtn, #tombol-buka, #tombolbuka')) {
        // Pause audio automatically after opening in edit mode
        setTimeout(function () {
          pauseAllAudio();
          notifyHost({ action: 'gate_opened' });
        }, 120);
        return;
      }

      // Allow our bg_photo floating pill
      if (target.closest('.dekorin-ve-bg-pill')) {
        return;
      }

      // Find nearest editable element
      var editable = target.closest('[data-edit-field]');
      if (!editable) return;

      // If editable is a large bg_photo container and user tapped inside non-editable card content,
      // only trigger bg_photo if they clicked outside form/buttons OR clicked directly on the background
      var editType = editable.getAttribute('data-edit-type');
      if (editType === 'bg_photo') {
        // Ignore clicks on RSVP form inputs/submit inside #ucapan
        if (target.closest('form, input, textarea, select, button:not(.dekorin-ve-bg-pill), a')) {
          return;
        }
      }

      e.preventDefault();
      e.stopPropagation();
      triggerEditElement(editable);
    },
    true
  );

  function pauseAllAudio() {
    try {
      document.querySelectorAll('audio').forEach(function (a) {
        a.pause();
      });
      var audioBtn = document.getElementById('ta17AudioBtn') || document.getElementById('ta3AudioBtn') || document.getElementById('mute-sound');
      if (audioBtn) audioBtn.classList.remove('playing');
    } catch (e) {}
  }

  // Re-parse Indonesian / ISO date string for PN4 split date & countdown
  function parseDateParts(raw) {
    if (!raw) return null;
    var s = String(raw).trim();
    if (!s) return null;
    var isoMatch = s.match(/(\d{4})-(\d{1,2})-(\d{1,2})/);
    var d = null;
    if (isoMatch) {
      d = new Date(parseInt(isoMatch[1], 10), parseInt(isoMatch[2], 10) - 1, parseInt(isoMatch[3], 10));
    } else {
      var monthsMap = {
        januari: 0, jan: 0, january: 0,
        februari: 1, feb: 1, february: 1,
        maret: 2, mar: 2, march: 2,
        april: 3, apr: 3,
        mei: 4, may: 4,
        juni: 5, jun: 5, june: 5,
        juli: 6, jul: 6, july: 6,
        agustus: 7, agu: 7, agt: 7, aug: 7, august: 7,
        september: 8, sep: 8, sept: 8,
        oktober: 9, okt: 9, oct: 9, october: 9,
        november: 10, nov: 10,
        desember: 11, des: 11, dec: 11, december: 11
      };
      var m = s.match(/(\d{1,2})\s+([A-Za-z]+)\s+(\d{4})/);
      if (m) {
        var mIdx = monthsMap[m[2].toLowerCase()];
        if (mIdx !== undefined) {
          d = new Date(parseInt(m[3], 10), mIdx, parseInt(m[1], 10));
        }
      }
    }
    if (!d || isNaN(d.getTime())) return null;
    var daysIndo = ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
    var monthsIndo = ['Januari', 'Februari', 'Maret', 'April', 'Mei', 'Juni', 'Juli', 'Agustus', 'September', 'Oktober', 'November', 'Desember'];
    return {
      dayName: daysIndo[d.getDay()],
      dayNum: String(d.getDate()),
      monthYear: monthsIndo[d.getMonth()] + ' ' + d.getFullYear(),
      dateObj: d
    };
  }

  function computeCoupleLabel() {
    var tpl = (state.template || '').toLowerCase();
    var isKhitan = (state.event_type || '').toLowerCase() === 'khitan';
    if (isKhitan) {
      return state.child_nickname || state.child_name || state.groom_nickname || state.groom_name || 'Putra Kami';
    }
    var g = state.groom_nickname || state.groom_name || 'Fulan';
    var b = state.bride_nickname || state.bride_name || 'Fulanah';
    // TA17 & TA3 put Bride first, PN4/Jawa put Groom first
    if (tpl === 'ta17' || tpl === 'ta3') {
      return b + ' & ' + g;
    }
    return g + ' & ' + b;
  }

  function formatChildOrderText(raw, defaultWord) {
    var s = (raw || '').trim();
    if (!s) return defaultWord + ' tercinta dari :';
    if (/^\d+$/.test(s)) {
      var ordinals = { '1': 'Pertama', '2': 'Kedua', '3': 'Ketiga', '4': 'Keempat', '5': 'Kelima' };
      return defaultWord + ' ' + (ordinals[s] || ('ke-' + s)) + ' Dari :';
    }
    if (s.toLowerCase().indexOf('dari') !== -1) return s;
    return s + ' dari';
  }

  function escapeHtml(str) {
    return String(str || '')
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  function parseLoveStoryItems(raw) {
    var s = String(raw || '').trim();
    if (!s) return [];
    if (s.charAt(0) === '[') {
      try {
        var parsed = JSON.parse(s);
        if (Array.isArray(parsed)) return parsed;
      } catch (e) {}
    }
    return s.split(/\r?\n/).map(function (line) {
      var trimmed = line.trim();
      if (!trimmed) return null;
      var colonIdx = trimmed.indexOf(':');
      var head = colonIdx > -1 ? trimmed.slice(0, colonIdx).trim() : trimmed;
      var desc = colonIdx > -1 ? trimmed.slice(colonIdx + 1).trim() : '';
      var dashParts = head.split(/\s+-\s+/);
      var date = dashParts.length > 1 ? dashParts[0].trim() : '';
      var title = dashParts.length > 1 ? dashParts.slice(1).join(' - ').trim() : head;
      return { date: date, title: title, description: desc };
    }).filter(Boolean);
  }

  function updateFieldDOM(key, value) {
    var tpl = (state.template || '').toLowerCase();

    // Update composite couple nicknames
    if (
      key === 'groom_nickname' ||
      key === 'bride_nickname' ||
      key === 'child_nickname' ||
      key === 'event_type'
    ) {
      var coupleLabel = computeCoupleLabel();
      document.querySelectorAll('[data-edit-field="couple_nick"]').forEach(function (el) {
        if (tpl === 'jawa3' && el.classList.contains('hero-couple-names') && (state.event_type || 'wedding') !== 'khitan') {
          var g = state.groom_nickname || state.groom_name || 'Putra';
          var b = state.bride_nickname || state.bride_name || 'Putri';
          el.innerHTML = escapeHtml(g) + '<br>&amp;<br>' + escapeHtml(b);
        } else {
          el.textContent = coupleLabel;
        }
      });
      var monoL = document.getElementById('monoLeft');
      var monoR = document.getElementById('monoRight');
      if (monoL && monoR) {
        var gn = (state.groom_nickname || state.groom_name || 'P').trim();
        var bn = (state.bride_nickname || state.bride_name || 'P').trim();
        monoL.textContent = gn ? gn.charAt(0).toUpperCase() : 'P';
        monoR.textContent = bn ? bn.charAt(0).toUpperCase() : 'P';
      }
    }

    // Standard direct elements matching data-edit-field="<key>"
    document.querySelectorAll('[data-edit-field="' + key + '"]').forEach(function (el) {
      var editType = el.getAttribute('data-edit-type');

      if (editType === 'photo') {
        if (value) {
          el.setAttribute('src', value);
          // For ta17 / ta3: switch illustration class to circle photo frame
          if (el.classList.contains('ta17-welcome-illus') || el.classList.contains('ta17-hero-illus') || el.classList.contains('ta17-person-illus') || el.classList.contains('ta17-closing-illus')) {
            el.classList.remove('ta17-welcome-illus', 'ta17-hero-illus', 'ta17-person-illus', 'ta17-closing-illus');
            el.classList.add('ta17-photo-circle');
          }
          if (el.classList.contains('ta3-welcome-illus') || el.classList.contains('ta3-hero-illus') || el.classList.contains('ta3-person-illus') || el.classList.contains('ta3-closing-illus')) {
            el.classList.remove('ta3-welcome-illus', 'ta3-hero-illus', 'ta3-person-illus', 'ta3-closing-illus');
            el.classList.add('ta3-photo-circle');
          }
        } else if (el.dataset.veInitialSrc) {
          el.setAttribute('src', el.dataset.veInitialSrc);
          if (el.dataset.veInitialClass) {
            el.className = el.dataset.veInitialClass;
          }
        }
        return;
      }

      if (editType === 'bg_photo') {
        if (value) {
          el.style.backgroundImage = "url('" + value.replace(/'/g, "\\'") + "')";
        } else if (el.dataset.veInitialBg) {
          el.style.backgroundImage = el.dataset.veInitialBg;
        }
        return;
      }

      if (editType === 'instagram') {
        var cleanIg = String(value || '').replace(/^@+/, '').trim();
        var span = el.querySelector('span');
        if (el.classList.contains('pn4-btn-dark')) {
          if (span) span.textContent = cleanIg || 'igmempelai';
        } else {
          if (span) span.textContent = cleanIg ? ('@' + cleanIg) : '@igmempelai';
        }
        if (cleanIg) {
          el.setAttribute('href', 'https://www.instagram.com/' + encodeURIComponent(cleanIg) + '/');
        }
        return;
      }

      if (editType === 'maps') {
        if (value) el.setAttribute('href', value);
        return;
      }

      if (editType === 'youtube') {
        if (value) el.setAttribute('href', value);
        return;
      }

      if (editType === 'child_order') {
        var roleWord = key.indexOf('bride') !== -1 ? 'Putri' : 'Putra';
        el.textContent = formatChildOrderText(value, roleWord);
        return;
      }

      if (key === 'turut_mengundang') {
        var ul = el.querySelector('ul');
        if (ul) {
          var lines = String(value || '').split(/\r?\n/).map(function (x) { return x.trim(); }).filter(Boolean);
          ul.innerHTML = lines.map(function (line) { return '<li>' + escapeHtml(line) + '</li>'; }).join('');
        }
        return;
      }

      if (editType === 'textarea' && key === 'quotes') {
        el.textContent = value || '';
        return;
      }

      el.textContent = value || '';
    });

    // Update PN4 split Akad / Resepsi dates & countdown data attributes
    if (key === 'akad_date') {
      var parsedAkad = parseDateParts(value);
      if (parsedAkad) {
        var akadDN = document.getElementById('akadDayName');
        var akadDNum = document.getElementById('akadDayNum');
        var akadMY = document.getElementById('akadMonthYear');
        if (akadDN) akadDN.textContent = parsedAkad.dayName;
        if (akadDNum) akadDNum.textContent = parsedAkad.dayNum;
        if (akadMY) akadMY.textContent = parsedAkad.monthYear;
      }
      document.body.setAttribute('data-akad-date', value || '');
      var cdBox = document.getElementById('countdownBox');
      if (cdBox) cdBox.setAttribute('data-date', value || '');
    }

    if (key === 'resepsi_date') {
      var parsedRes = parseDateParts(value);
      if (parsedRes) {
        var resDN = document.getElementById('resepsiDayName');
        var resDNum = document.getElementById('resepsiDayNum');
        var resMY = document.getElementById('resepsiMonthYear');
        if (resDN) resDN.textContent = parsedRes.dayName;
        if (resDNum) resDNum.textContent = parsedRes.dayNum;
        if (resMY) resMY.textContent = parsedRes.monthYear;
      }
      document.body.setAttribute('data-resepsi-date', value || '');
    }

    // Update bank cards live
    if (key.indexOf('bank_') === 0) {
      [1, 2].forEach(function (idx) {
        var bCard = document.querySelector('[data-edit-field="bank_' + idx + '"]');
        if (!bCard) return;
        var bName = state['bank_name_' + idx] || '';
        var bAcc = state['bank_account_' + idx] || '';
        var bHold = state['bank_holder_' + idx] || '';

        var lbl = bCard.querySelector('.ta17-gift-label, .ta3-gift-label');
        var valEl = bCard.querySelector('.ta17-gift-value, .ta3-gift-value');
        if (lbl) lbl.textContent = 'transfer ke rekening ' + bName + ' a.n ' + bHold;
        if (valEl) valEl.textContent = bAcc;

        var pnHolder = bCard.querySelector('.pn4-bank-holder');
        var pnAcc = bCard.querySelector('.pn4-bank-acc');
        if (pnHolder) pnHolder.textContent = bHold;
        if (pnAcc) pnAcc.textContent = bAcc;

        var jawaInfo = bCard.querySelector('.bank-info-text');
        if (jawaInfo) {
          jawaInfo.innerHTML = escapeHtml(bName) + '<br>No. Rekening ' + escapeHtml(bAcc) + '<br>a.n <strong>' + escapeHtml(bHold) + '</strong>';
        }
        var j3Hold = bCard.querySelector('.bank-holder');
        var j3Num = bCard.querySelector('.bank-number');
        if (j3Hold) j3Hold.textContent = bHold;
        if (j3Num) j3Num.textContent = bAcc;
      });
    }

    // Update gallery grid live
    if (key === 'gallery_images') {
      var urls = splitGallery(value);
      var taGrid = document.getElementById('ta17GalleryGrid') || document.getElementById('ta3GalleryGrid');
      if (taGrid) {
        var cls = taGrid.id.indexOf('ta3') !== -1 ? 'ta3-masonry-item' : 'ta17-masonry-item';
        taGrid.innerHTML = urls.map(function (u, i) {
          return '<div class="' + cls + '" data-idx="' + i + '"><img src="' + escapeHtml(u) + '" alt="Gallery ' + i + '"></div>';
        }).join('');
      }
      var pnGrid = document.getElementById('pn4JustifiedGallery');
      if (pnGrid) {
        pnGrid.innerHTML = urls.map(function (u, i) {
          return '<div class="pn4-jgal-item" style="flex:1 1 140px;height:180px;"><img src="' + escapeHtml(u) + '" alt="Gallery ' + i + '" style="width:100%;height:100%;object-fit:cover;border-radius:10px;"></div>';
        }).join('');
      }
      var jawaGrid = document.querySelector('#galeri .gallery-grid');
      if (jawaGrid) {
        jawaGrid.innerHTML = urls.map(function (u, i) {
          return '<div class="gallery-item"><img src="' + escapeHtml(u) + '" alt="Galeri ' + i + '"></div>';
        }).join('');
      }
    }

    // Update Love Story live
    if (key === 'love_story') {
      var items = parseLoveStoryItems(value);
      var taTimeline = document.getElementById('ta17Timeline') || document.getElementById('ta3Timeline');
      if (taTimeline) {
        var prefix = taTimeline.id.indexOf('ta3') !== -1 ? 'ta3' : 'ta17';
        var lineEl = taTimeline.querySelector('.' + prefix + '-timeline-line');
        var itemsHtml = items.map(function (it) {
          return (
            '<div class="' + prefix + '-tl-item">' +
            '  <div class="' + prefix + '-tl-dot"></div>' +
            '  <div class="' + prefix + '-tl-card">' +
            (it.date ? '<div class="' + prefix + '-tl-date">' + escapeHtml(it.date) + '</div>' : '') +
            (it.title ? '<h4 class="' + prefix + '-tl-title">' + escapeHtml(it.title) + '</h4>' : '') +
            (it.description ? '<p class="' + prefix + '-tl-desc">' + escapeHtml(it.description) + '</p>' : '') +
            '  </div>' +
            '</div>'
          );
        }).join('');
        taTimeline.innerHTML = '';
        if (lineEl) taTimeline.appendChild(lineEl);
        taTimeline.insertAdjacentHTML('beforeend', itemsHtml);
      }

      var pnStory = document.getElementById('pn4StoryList');
      if (pnStory) {
        pnStory.innerHTML = items.map(function (it) {
          return (
            '<div class="pn4-story-card pn-anim anim-zoomIn is-visible">' +
            (it.date ? '<span class="pn4-story-date">' + escapeHtml(it.date) + '</span>' : '') +
            (it.title ? '<span class="pn4-story-title">' + escapeHtml(it.title) + '</span>' : '') +
            (it.description ? '<p class="pn4-story-body">' + escapeHtml(it.description) + '</p>' : '') +
            '</div>'
          );
        }).join('');
      }

      var jawa2Story = document.getElementById('loveStoryTimelineContainer');
      if (jawa2Story) {
        jawa2Story.innerHTML = items.map(function (it) {
          return (
            '<div class="ls-item">' +
            '  <div class="ls-card">' +
            (it.date ? '<div class="ls-date">' + escapeHtml(it.date) + '</div>' : '') +
            (it.title ? '<div class="ls-title">' + escapeHtml(it.title) + '</div>' : '') +
            (it.description ? '<div class="ls-desc">' + escapeHtml(it.description) + '</div>' : '') +
            '  </div>' +
            '</div>'
          );
        }).join('');
      }

      var jawa3Slides = document.getElementById('loveStorySlidesContainer');
      if (jawa3Slides) {
        jawa3Slides.innerHTML = items.map(function (it, idx) {
          return (
            '<div class="ls-slide' + (idx === 0 ? ' active' : '') + '">' +
            (it.date ? '<div class="ls-date">' + escapeHtml(it.date) + '</div>' : '') +
            (it.title ? '<div class="ls-title">' + escapeHtml(it.title) + '</div>' : '') +
            (it.description ? '<div class="ls-desc">' + escapeHtml(it.description) + '</div>' : '') +
            '</div>'
          );
        }).join('');
      }
    }

    refreshEmptyPlaceholderSummaries();
  }

  window.dekorinApplyPatch = function (patch) {
    if (!patch || typeof patch !== 'object') return;
    Object.keys(patch).forEach(function (k) {
      state[k] = patch[k];
    });
    Object.keys(patch).forEach(function (k) {
      updateFieldDOM(k, patch[k]);
    });
  };

  window.dekorinSetEditMode = setEditMode;

  window.dekorinToggleGate = function (showGate) {
    // TA17 & TA3
    var taGate = document.getElementById('ta17ModalGate') || document.getElementById('ta3ModalGate');
    if (taGate) {
      if (showGate) {
        taGate.classList.remove('closed');
        document.body.classList.add('scroll-locked');
      } else {
        taGate.classList.add('closed');
        document.body.classList.remove('scroll-locked');
      }
      return;
    }
    // Jawa2 & Jawa3
    var jawaCover = document.getElementById('cover');
    if (jawaCover && !document.getElementById('gate-cover')) {
      if (showGate) {
        jawaCover.classList.remove('closed');
        document.body.classList.add('locked');
      } else {
        if (typeof window.openInvitation === 'function') {
          window.openInvitation();
        } else {
          jawaCover.classList.add('closed');
          document.body.classList.remove('locked');
        }
        setTimeout(pauseAllAudio, 120);
      }
      return;
    }
    // PN4
    if (document.getElementById('gate-cover')) {
      document.body.classList.remove('scroll-locked');
      if (showGate) {
        window.scrollTo({ top: 0, behavior: 'smooth' });
      } else {
        var cov = document.getElementById('cover');
        if (cov) cov.scrollIntoView({ behavior: 'smooth' });
      }
    }
  };

  // Listen to postMessage from Web Portal parent iframe
  window.addEventListener('message', function (ev) {
    var msg = ev && ev.data;
    if (!msg || typeof msg !== 'object') return;
    if (msg.source !== 'dekorin-portal-parent') return;
    if (msg.action === 'apply_patch' && msg.patch) {
      window.dekorinApplyPatch(msg.patch);
    } else if (msg.action === 'set_edit_mode') {
      setEditMode(msg.enabled);
    } else if (msg.action === 'toggle_gate') {
      window.dekorinToggleGate(!!msg.showGate);
    }
  });

  function captureInitialPhotos() {
    document.querySelectorAll('img[data-edit-field]').forEach(function (el) {
      el.dataset.veInitialSrc = el.getAttribute('src') || '';
      el.dataset.veInitialClass = el.className || '';
    });
    document.querySelectorAll('[data-edit-type="bg_photo"]').forEach(function (el) {
      el.dataset.veInitialBg = el.style.backgroundImage || '';
    });
  }

  function initVisualEditor() {
    document.body.classList.add('dekorin-ve-active');
    captureInitialPhotos();
    var amplop = document.getElementById('amplop');
    if (amplop) amplop.style.display = 'block';
    ensureBgPhotoBadges();
    attachEmptyPlaceholders();
    refreshEmptyPlaceholderSummaries();
    pauseAllAudio();
    notifyHost({ action: 'ready', template: state.template || '' });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initVisualEditor);
  } else {
    initVisualEditor();
  }
})();
