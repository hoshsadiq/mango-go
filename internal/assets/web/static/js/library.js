import { checkAuth } from './auth.js';

document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth();
  if (!currentUser) return;

  const cardsGrid = document.getElementById('cards-grid');
  const pageTitleEl = document.getElementById('page-title');
  const breadcrumbEl = document.getElementById('breadcrumb-container');
  const folderThumb = document.getElementById('folder-thumb');
  const searchInput = document.getElementById('search-input');
  const sortBySelect = document.getElementById('sort-by');
  const sortDirBtn = document.getElementById('sort-dir-btn');
  const paginationContainer = document.getElementById('pagination-container');
  const editFolderBtn = document.getElementById('edit-folder-btn');
  const editFolderModal = document.getElementById('edit-folder-modal');
  const modalCloseBtn = document.getElementById('modal-close-btn');
  const tagsContainer = document.getElementById('tags-container');
  const tagInput = document.getElementById('tag-input');
  const autocompleteSuggestions = document.getElementById('autocomplete-suggestions');
  const modalSaveBtn = document.getElementById('modal-save-btn');
  const modalCancelBtn = document.getElementById('modal-cancel-btn');
  const coverFileInput = document.getElementById('cover-file-input');
  const selectedFileInfo = document.getElementById('selected-file-info');
  const fileName = document.getElementById('file-name');
  const removeFileBtn = document.getElementById('remove-file-btn');
  const totalCountEl = document.getElementById('total-count');
  const markAllReadBtn = document.getElementById('mark-all-read-btn');
  const markAllUnreadBtn = document.getElementById('mark-all-unread-btn');
  const unreadFilterBtn = document.getElementById('unread-filter-btn');
  const tagFilterBar = document.getElementById('tag-filter-bar');
  const tagFilterChips = document.getElementById('tag-filter-chips');
  const ratingWidget = document.getElementById('rating-widget');
  const ratingStars = document.getElementById('rating-stars');
  const ratingClearBtn = document.getElementById('rating-clear-btn');
  const progressActions = document.getElementById('progress-actions');
  const folderTagsSection = document.getElementById('folder-tags-section');
  const metadataPanel = document.getElementById('metadata-panel');
  const noMetadataPrompt = document.getElementById('no-metadata-prompt');
  const mdStatusBadge = document.getElementById('md-status-badge');
  const mdHeaderActions = document.getElementById('md-header-actions');
  const communityScoreWidget = document.getElementById('community-score-widget');
  const csValue = document.getElementById('cs-value');

  const mdEditBtn = document.getElementById('md-edit-btn');
  const mdRefreshBtn = document.getElementById('md-refresh-btn');
  const mdRelinkBtn = document.getElementById('md-relink-btn');
  const mdResetBtn = document.getElementById('md-reset-btn');
  const mdUnlinkBtn = document.getElementById('md-unlink-btn');
  const metadataSearchModal = document.getElementById('metadata-search-modal');
  const mdSearchInput = document.getElementById('md-search-input');
  const mdSearchBtn = document.getElementById('md-search-btn');
  const mdSearchCloseBtn = document.getElementById('md-search-close-btn');
  const mdSearchCancelBtn = document.getElementById('md-search-cancel-btn');
  const mdSearchResults = document.getElementById('md-search-results');
  const mdSearchLoading = document.getElementById('md-search-loading');
  const mdSearchError = document.getElementById('md-search-error');
  const mdLinkBtn = document.getElementById('md-link-btn');
  const linkMetadataBtn = document.getElementById('link-metadata-btn');

  // --- State Management ---
  let state = {
    currentFolderId: null,
    currentTagId: null,
    filterTagId: null,
    currentPage: 1,
    search: '',
    sortBy: null,
    sortDir: null,
    unreadOnly: localStorage.getItem('unreadOnly') === 'true',
    isLoading: false,
    totalItems: 0,
    perPage: 100,
    currentRating: null,
  };
  let allTags = [];
  let currentFolderTags = [];
  let tagsExpanded = false;
  let filterChipsExpanded = false;
  let mdSelectedResult = null;
  let mdOriginalMetadata = null;
  let mdOriginalProvider = null;
  let mdEditMode = false;
  let originalFolderName = '';
  const TAGS_COLLAPSED_LIMIT = 8;
  const FILTER_CHIPS_LIMIT = 10;

  // --- Core Functions ---

  // AniList: load from DB first, then trigger server fetch on cache miss
  const fetchAniListData = async folderId => {
    if (!folderId) return null;
    try {
      const getRes = await fetch(`/api/folders/${folderId}/anilist`);
      if (getRes.ok) {
        const data = await getRes.json();
        if (data?.siteUrl) return data;
      }
      const postRes = await fetch(`/api/folders/${folderId}/anilist`, { method: 'POST' });
      if (!postRes.ok) return null;
      const data = await postRes.json();
      return data?.siteUrl ? data : null;
    } catch (error) {
      console.error('Error fetching AniList data for folder:', folderId, error);
      return null;
    }
  };

  // Async AniList button loader - runs after page load without blocking
  const loadAniListButtonsAsync = () => {
    // Use setTimeout to ensure this runs after the main rendering is complete
    setTimeout(async () => {
      // Skip if button already exists
      if (document.querySelector('.anilist-button')) return;
      const folderId = getFolderIdFromUrl();
      if (!folderId) return; // Only show AniList button when viewing a specific folder

      try {
        const anilistData = await fetchAniListData(folderId);
        if (anilistData?.siteUrl) {
          addAniListButtonToHeader(anilistData.siteUrl);
        }
      } catch (error) {
        console.warn('Failed to load AniList data for folder:', folderId, error);
      }
    }, 100); // Small delay to ensure DOM is ready
  };

  // Appends the AniList link button into the existing header-actions row.
  // Inserting here (rather than wrapping the h1) avoids breaking the flex layout.
  const addAniListButtonToHeader = anilistUrl => {
    // Guard: skip if the container is missing/hidden or the button was already added
    if (!mdHeaderActions || document.querySelector('.anilist-button')) return;

    const button = document.createElement('a');
    button.href = anilistUrl;
    button.target = '_blank';
    button.className = 'anilist-button md-icon-btn';
    button.title = 'View on AniList';

    const icon = document.createElement('img');
    icon.src = '/static/images/anilist-icon.svg';
    icon.alt = 'AniList';
    icon.className = 'anilist-icon';

    button.appendChild(icon);
    mdHeaderActions.appendChild(button);
  };

  const escapeHtml = str => {
    if (!str) return '';
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  };

  const formatReadingDirection = val => {
    const map = {
      LEFT_TO_RIGHT: 'Left to Right',
      RIGHT_TO_LEFT: 'Right to Left',
      VERTICAL: 'Vertical',
      WEBTOON: 'Webtoon',
    };
    return map[val] || val;
  };

  const formatAuthorRole = role => {
    return role
      .split('_')
      .map(w => w.charAt(0) + w.slice(1).toLowerCase())
      .join(' ');
  };

  const formatReleaseDate = (year, month, day) => {
    if (!year) return null;
    let date = String(year);
    if (month) {
      date += '-' + String(month).padStart(2, '0');
      if (day) {
        date += '-' + String(day).padStart(2, '0');
      }
    }
    return date;
  };

  // Lock icon helper, now always renders an icon (clickable toggle).
  const lockIconHtml = (locked, lockField) => {
    if (locked) {
      return ` <i class="ph-bold ph-lock md-lock md-lock-toggle" data-lock-field="${lockField}" data-locked="true" title="Locked: this field won't be changed on refresh"></i>`;
    }
    return ` <i class="ph-bold ph-lock-open md-lock md-lock-toggle md-lock-unlocked" data-lock-field="${lockField}" data-locked="false" title="Unlocked: this field will be updated on refresh"></i>`;
  };
  // Map from lock field name to the lock response key (without _lock suffix)
  const lockFieldToResponseKey = lockField => {
    return lockField.replace(/_lock$/, '');
  };

  const toggleFieldLock = async (folderId, lockField, currentlyLocked) => {
    const newValue = !currentlyLocked;
    try {
      const res = await fetch(`/api/folders/${folderId}/metadata/locks`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ [lockField]: newValue }),
      });
      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        throw new Error(errData.error || `Lock toggle failed (HTTP ${res.status})`);
      }
      const locks = await res.json();
      if (mdOriginalMetadata) {
        mdOriginalMetadata.locks = locks;
      }
      const icon = document.querySelector(`.md-lock-toggle[data-lock-field="${lockField}"]`);
      if (icon) {
        const isLocked = locks[lockFieldToResponseKey(lockField)];
        icon.dataset.locked = String(isLocked);
        icon.className = isLocked
          ? 'ph-bold ph-lock md-lock md-lock-toggle'
          : 'ph-bold ph-lock-open md-lock md-lock-toggle md-lock-unlocked';
        icon.title = isLocked
          ? "Locked: this field won't be changed on refresh"
          : 'Unlocked: this field will be updated on refresh';
      }
    } catch (err) {
      toast.error(err.message);
    }
  };

  const attachLockListeners = folderId => {
    document.querySelectorAll('.md-lock-toggle').forEach(icon => {
      icon.addEventListener('click', e => {
        e.stopPropagation();
        const lockField = icon.dataset.lockField;
        const currentlyLocked = icon.dataset.locked === 'true';
        toggleFieldLock(folderId, lockField, currentlyLocked);
      });
    });
  };

  // --- Validation helpers ---
  const validateNumericField = (value, fieldName) => {
    if (value === '' || value === null || value === undefined) return null; // empty = clear
    const num = Number(value);
    if (isNaN(num)) return `${fieldName} must be a number`;
    switch (fieldName) {
      case 'age_rating':
        if (num < 0 || !Number.isInteger(num)) return 'Age rating must be a non-negative integer';
        break;
      case 'community_score':
        if (num < 0 || num > 10) return 'Score must be between 0.0 and 10.0';
        break;
      case 'release_year':
        if (num < 1000 || num > 9999 || !Number.isInteger(num))
          return 'Year must be a 4-digit number';
        break;
      case 'release_month':
        if (num < 1 || num > 12 || !Number.isInteger(num)) return 'Month must be between 1 and 12';
        break;
      case 'release_day':
        if (num < 1 || num > 31 || !Number.isInteger(num)) return 'Day must be between 1 and 31';
        break;
      case 'total_book_count':
        if (!Number.isInteger(num)) return 'Volume count must be an integer';
        break;
    }
    return null;
  };

  const showFieldError = (inputEl, message) => {
    inputEl.classList.add('md-input-error');
    let errEl = inputEl.parentElement.querySelector('.md-field-error');
    if (!errEl) {
      errEl = document.createElement('div');
      errEl.className = 'md-field-error';
      inputEl.parentElement.appendChild(errEl);
    }
    errEl.textContent = message;
    errEl.style.display = 'block';
  };

  const clearFieldError = inputEl => {
    inputEl.classList.remove('md-input-error');
    const errEl = inputEl.parentElement.querySelector('.md-field-error');
    if (errEl) errEl.style.display = 'none';
  };


  // --- Authors edit helpers ---
  const AUTHOR_ROLES = [
    'WRITER',
    'PENCILLER',
    'INKER',
    'COLORIST',
    'LETTERER',
    'COVER_ARTIST',
    'EDITOR',
    'TRANSLATOR',
  ];

  const renderAuthorRow = (author, idx) => {
    let html = `<div class="md-author-edit-row" data-idx="${idx}">`;
    html += `<input type="text" class="md-edit-input md-author-name-input" value="${escapeHtml(author.name)}" placeholder="Name">`;
    html += `<select class="md-edit-select md-author-role-select">`;
    AUTHOR_ROLES.forEach(r => {
      html += `<option value="${r}"${r === author.role ? ' selected' : ''}>${escapeHtml(formatAuthorRole(r))}</option>`;
    });
    html += `</select>`;
    html += `<button class="md-row-remove" type="button" title="Remove">&times;</button>`;
    html += '</div>';
    return html;
  };

  // --- Links edit helpers ---
  const renderLinkRow = (link, idx) => {
    let html = `<div class="md-link-edit-row" data-idx="${idx}">`;
    html += `<input type="text" class="md-edit-input md-link-label-input" value="${escapeHtml(link.label)}" placeholder="Label">`;
    html += `<input type="url" class="md-edit-input md-link-url-input" value="${escapeHtml(link.url)}" placeholder="URL">`;
    html += `<button class="md-row-remove" type="button" title="Remove">&times;</button>`;
    html += '</div>';
    return html;
  };

  // --- Alt titles edit helpers ---
  const TITLE_TYPES = ['ROMAJI', 'LOCALIZED', 'NATIVE'];

  const renderAltTitleRow = (title, idx) => {
    let html = `<div class="md-title-edit-row" data-idx="${idx}">`;
    html += `<input type="text" class="md-edit-input md-title-text-input" value="${escapeHtml(title.title)}" placeholder="Title">`;
    html += `<select class="md-edit-select md-title-type-select">`;
    TITLE_TYPES.forEach(t => {
      html += `<option value="${t}"${t === title.type ? ' selected' : ''}>${t}</option>`;
    });
    html += `</select>`;
    html += `<input type="text" class="md-edit-input md-title-lang-input" value="${escapeHtml(title.language || '')}" placeholder="Language">`;
    html += `<button class="md-row-remove" type="button" title="Remove">&times;</button>`;
    html += '</div>';
    return html;
  };

  const buildMetadataPanelHtml = (md, locks, provider, editMode) => {
    let html = '';

    if (editMode) {
      html += '<div class="md-header">';
      html += '<span class="md-actions-spacer"></span>';
      html += `<button class="md-action-btn md-save-btn" id="md-save-btn" title="Edited fields are automatically locked to prevent provider overwrite"><i class="ph-bold ph-floppy-disk"></i> Save</button>`;
      html += `<button class="md-action-btn" id="md-cancel-btn"><i class="ph-bold ph-x"></i> Cancel</button>`;
      html += '</div>';
      html +=
        '<div class="md-edit-save-note"><i class="ph-bold ph-info"></i> Edited fields are automatically locked to prevent provider overwrite.</div>';
    }

    if (editMode) {
      html += `<div class="md-section-label">Alternative Titles${lockIconHtml(locks.titles, 'titles_lock')}</div>`;
      html += '<div class="md-edit-collection" id="md-edit-titles">';
      (md.titles || []).forEach((t, i) => {
        html += renderAltTitleRow(t, i);
      });
      html += `<button class="md-add-row-btn" id="md-add-title-btn" type="button"><i class="ph-bold ph-plus"></i> Add Title</button>`;
      html += `<div class="md-edit-note">Collection editing not yet supported by the API — changes will not be saved.</div>`;
      html += '</div>';
    } else if (md.titles && md.titles.length > 0) {
      html += '<div class="md-alt-titles">';
      html += `<button class="md-alt-titles-toggle" data-expanded="false">`;
      html += `<i class="ph-bold ph-caret-right"></i> ${md.titles.length} Alternative Title${md.titles.length > 1 ? 's' : ''}${lockIconHtml(locks.titles, 'titles_lock')}`;
      html += `</button>`;
      html += '<ul class="md-alt-titles-list">';
      md.titles.forEach(t => {
        const lang = t.language
          ? ` <span class="md-title-lang">(${escapeHtml(t.language)})</span>`
          : '';
        html += `<li>${escapeHtml(t.title)}${lang}</li>`;
      });
      html += '</ul></div>';
    }

    if (editMode || (md.summary && md.summary.trim())) {
      html += `<div class="md-summary">`;
      html += `<div class="md-section-label">Summary${lockIconHtml(locks.summary, 'summary_lock')}</div>`;
      if (editMode) {
        html += `<textarea class="md-edit-textarea" data-field="summary" rows="4" placeholder="Summary">${escapeHtml(md.summary || '')}</textarea>`;
      } else {
        html += `<p class="md-summary-text collapsed">${escapeHtml(md.summary)}</p>`;
        html += `<button class="md-summary-toggle" data-expanded="false">Show more</button>`;
      }
      html += '</div>';
    }


    html += '<div class="md-info-grid">';

    if (editMode) {
      html += `<div class="md-info-item"><span class="md-info-label">Publisher${lockIconHtml(locks.publisher, 'publisher_lock')}</span>`;
      html += `<input type="text" class="md-edit-input" data-field="publisher" value="${escapeHtml(md.publisher || '')}"></div>`;
    } else if (md.publisher) {
      html += `<div class="md-info-item"><span class="md-info-label">Publisher</span><span class="md-info-value">${escapeHtml(md.publisher)}</span>${lockIconHtml(locks.publisher, 'publisher_lock')}</div>`;
    }

    if (editMode) {
      html += `<div class="md-info-item"><span class="md-info-label">Direction${lockIconHtml(locks.reading_direction, 'reading_direction_lock')}</span>`;
      html += `<select class="md-edit-select" data-field="reading_direction">`;
      html += `<option value=""${!md.reading_direction ? ' selected' : ''}>— None —</option>`;
      ['LEFT_TO_RIGHT', 'RIGHT_TO_LEFT', 'VERTICAL', 'WEBTOON'].forEach(d => {
        html += `<option value="${d}"${md.reading_direction === d ? ' selected' : ''}>${formatReadingDirection(d)}</option>`;
      });
      html += `</select></div>`;
    } else if (md.reading_direction) {
      html += `<div class="md-info-item"><span class="md-info-label">Direction</span><span class="md-info-value">${escapeHtml(formatReadingDirection(md.reading_direction))}</span>${lockIconHtml(locks.reading_direction, 'reading_direction_lock')}</div>`;
    }

    if (editMode) {
      html += `<div class="md-info-item"><span class="md-info-label">Age Rating${lockIconHtml(locks.age_rating, 'age_rating_lock')}</span>`;
      html += `<input type="number" class="md-edit-input md-edit-number" data-field="age_rating" min="0" step="1" value="${md.age_rating !== null && md.age_rating !== undefined ? md.age_rating : ''}" placeholder="e.g. 13"></div>`;
    } else if (md.age_rating !== null && md.age_rating !== undefined) {
      html += `<div class="md-info-item"><span class="md-info-label">Age Rating</span><span class="md-info-value">${md.age_rating}+</span>${lockIconHtml(locks.age_rating, 'age_rating_lock')}</div>`;
    }

    if (editMode) {
      html += `<div class="md-info-item"><span class="md-info-label">Language${lockIconHtml(locks.language, 'language_lock')}</span>`;
      html += `<input type="text" class="md-edit-input" data-field="language" value="${escapeHtml(md.language || '')}" placeholder="e.g. ja"></div>`;
    } else if (md.language) {
      html += `<div class="md-info-item"><span class="md-info-label">Language</span><span class="md-info-value">${escapeHtml(md.language)}</span>${lockIconHtml(locks.language, 'language_lock')}</div>`;
    }

    if (editMode) {
      html += `<div class="md-info-item"><span class="md-info-label">Volumes${lockIconHtml(locks.total_book_count, 'total_book_count_lock')}</span>`;
      html += `<input type="number" class="md-edit-input md-edit-number" data-field="total_book_count" step="1" value="${md.total_book_count !== null && md.total_book_count !== undefined ? md.total_book_count : ''}" placeholder="e.g. 10"></div>`;
    } else if (md.total_book_count !== null && md.total_book_count !== undefined) {
      html += `<div class="md-info-item"><span class="md-info-label">Volumes</span><span class="md-info-value">${md.total_book_count}</span>${lockIconHtml(locks.total_book_count, 'total_book_count_lock')}</div>`;
    }

    if (editMode) {
      html += `<div class="md-info-item md-release-date-edit"><span class="md-info-label">Released${lockIconHtml(locks.release_date, 'release_date_lock')}</span>`;
      html += `<input type="number" class="md-edit-input md-edit-number md-date-input" data-field="release_year" min="1000" max="9999" step="1" value="${md.release_year !== null && md.release_year !== undefined ? md.release_year : ''}" placeholder="Year">`;
      html += `<input type="number" class="md-edit-input md-edit-number md-date-input" data-field="release_month" min="1" max="12" step="1" value="${md.release_month !== null && md.release_month !== undefined ? md.release_month : ''}" placeholder="Mo">`;
      html += `<input type="number" class="md-edit-input md-edit-number md-date-input" data-field="release_day" min="1" max="31" step="1" value="${md.release_day !== null && md.release_day !== undefined ? md.release_day : ''}" placeholder="Day">`;
      html += `</div>`;
    } else {
      const releaseDate = formatReleaseDate(md.release_year, md.release_month, md.release_day);
      if (releaseDate) {
        html += `<div class="md-info-item"><span class="md-info-label">Released</span><span class="md-info-value">${escapeHtml(releaseDate)}</span>${lockIconHtml(locks.release_date, 'release_date_lock')}</div>`;
      }
    }

    html += '</div>';

    if (editMode) {
      html += '<div class="md-score">';
      html += `<span class="md-section-label">Community Score${lockIconHtml(locks.community_score, 'community_score_lock')}</span>`;
      html += `<input type="number" class="md-edit-input md-edit-number" data-field="community_score" min="0" max="10" step="0.1" value="${md.community_score !== null && md.community_score !== undefined ? md.community_score : ''}" placeholder="0.0 - 10.0">`;
      html += '</div>';
    }

    if (editMode || (md.authors && md.authors.length > 0)) {
      html += `<div class="md-section-label">Authors${lockIconHtml(locks.authors, 'authors_lock')}</div>`;
      if (editMode) {
        html += '<div class="md-edit-collection" id="md-edit-authors">';
        (md.authors || []).forEach((a, i) => {
          html += renderAuthorRow(a, i);
        });
        html += `<button class="md-add-row-btn" id="md-add-author-btn" type="button"><i class="ph-bold ph-plus"></i> Add Author</button>`;
        html += `<div class="md-edit-note">Collection editing not yet supported by the API — changes will not be saved.</div>`;
        html += '</div>';
      } else {
        html += '<div class="md-authors">';
        const groups = {};
        md.authors.forEach(a => {
          const role = a.role || 'Other';
          if (!groups[role]) groups[role] = [];
          groups[role].push(a.name);
        });
        Object.entries(groups).forEach(([role, names]) => {
          html += '<div class="md-author-group">';
          html += `<div class="md-author-role">${escapeHtml(formatAuthorRole(role))}</div>`;
          names.forEach(name => {
            html += `<div class="md-author-name">${escapeHtml(name)}</div>`;
          });
          html += '</div>';
        });
        html += '</div>';
      }
    }

    if (editMode || (md.links && md.links.length > 0)) {
      html += `<div class="md-section-label">Links${lockIconHtml(locks.links, 'links_lock')}</div>`;
      if (editMode) {
        html += '<div class="md-edit-collection" id="md-edit-links">';
        (md.links || []).forEach((l, i) => {
          html += renderLinkRow(l, i);
        });
        html += `<button class="md-add-row-btn" id="md-add-link-btn" type="button"><i class="ph-bold ph-plus"></i> Add Link</button>`;
        html += `<div class="md-edit-note">Collection editing not yet supported by the API — changes will not be saved.</div>`;
        html += '</div>';
      } else {
        html += '<div class="md-links">';
        md.links.forEach(link => {
          html += `<a class="md-ext-link" href="${escapeHtml(link.url)}" target="_blank" rel="noopener noreferrer">`;
          html += `<i class="ph-bold ph-arrow-square-out"></i> ${escapeHtml(link.label)}`;
          html += '</a>';
        });
        html += '</div>';
      }
    }

    return html;
  };

  const attachPanelListeners = (folderId, editMode) => {
    attachLockListeners(folderId);

    if (editMode) {
      // Save button
      const saveBtn = metadataPanel.querySelector('#md-save-btn');
      if (saveBtn) saveBtn.addEventListener('click', () => handleMetadataSave(folderId));

      // Cancel button
      const cancelBtn = metadataPanel.querySelector('#md-cancel-btn');
      if (cancelBtn) cancelBtn.addEventListener('click', handleMetadataCancel);

      // Add author row
      const addAuthorBtn = metadataPanel.querySelector('#md-add-author-btn');
      if (addAuthorBtn) {
        addAuthorBtn.addEventListener('click', () => {
          const container = metadataPanel.querySelector('#md-edit-authors');
          const rows = container.querySelectorAll('.md-author-edit-row');
          const idx = rows.length;
          const newRowHtml = renderAuthorRow({ name: '', role: 'WRITER' }, idx);
          addAuthorBtn.insertAdjacentHTML('beforebegin', newRowHtml);
          const newRow = container.querySelectorAll('.md-author-edit-row')[idx];
          newRow.querySelector('.md-row-remove').addEventListener('click', () => newRow.remove());
        });
      }

      // Add link row
      const addLinkBtn = metadataPanel.querySelector('#md-add-link-btn');
      if (addLinkBtn) {
        addLinkBtn.addEventListener('click', () => {
          const container = metadataPanel.querySelector('#md-edit-links');
          const rows = container.querySelectorAll('.md-link-edit-row');
          const idx = rows.length;
          const newRowHtml = renderLinkRow({ label: '', url: '' }, idx);
          addLinkBtn.insertAdjacentHTML('beforebegin', newRowHtml);
          const newRow = container.querySelectorAll('.md-link-edit-row')[idx];
          newRow.querySelector('.md-row-remove').addEventListener('click', () => newRow.remove());
        });
      }

      // Add alt title row
      const addTitleBtn = metadataPanel.querySelector('#md-add-title-btn');
      if (addTitleBtn) {
        addTitleBtn.addEventListener('click', () => {
          const container = metadataPanel.querySelector('#md-edit-titles');
          const rows = container.querySelectorAll('.md-title-edit-row');
          const idx = rows.length;
          const newRowHtml = renderAltTitleRow({ title: '', type: 'ROMAJI', language: '' }, idx);
          addTitleBtn.insertAdjacentHTML('beforebegin', newRowHtml);
          const newRow = container.querySelectorAll('.md-title-edit-row')[idx];
          newRow.querySelector('.md-row-remove').addEventListener('click', () => newRow.remove());
        });
      }

      // Remove row buttons for existing rows
      metadataPanel.querySelectorAll('.md-row-remove').forEach(btn => {
        btn.addEventListener('click', () => btn.closest('[data-idx]').remove());
      });

      // Validation on numeric inputs
      metadataPanel.querySelectorAll('.md-edit-number').forEach(input => {
        input.addEventListener('input', () => {
          const field = input.dataset.field;
          const err = validateNumericField(input.value, field);
          if (err) {
            showFieldError(input, err);
          } else {
            clearFieldError(input);
          }
        });
      });
    } else {
      const altTitlesToggle = metadataPanel.querySelector('.md-alt-titles-toggle');
      if (altTitlesToggle) {
        altTitlesToggle.addEventListener('click', () => {
          const list = metadataPanel.querySelector('.md-alt-titles-list');
          const expanded = altTitlesToggle.dataset.expanded === 'true';
          altTitlesToggle.dataset.expanded = String(!expanded);
          list.classList.toggle('expanded', !expanded);
          const icon = altTitlesToggle.querySelector('i');
          icon.className = !expanded ? 'ph-bold ph-caret-down' : 'ph-bold ph-caret-right';
        });
      }

      const summaryToggle = metadataPanel.querySelector('.md-summary-toggle');
      if (summaryToggle) {
        const summaryText = metadataPanel.querySelector('.md-summary-text');
        setTimeout(() => {
          if (summaryText && summaryText.scrollHeight <= summaryText.clientHeight) {
            summaryToggle.style.display = 'none';
          }
        }, 0);
        summaryToggle.addEventListener('click', () => {
          const expanded = summaryToggle.dataset.expanded === 'true';
          summaryToggle.dataset.expanded = String(!expanded);
          summaryText.classList.toggle('collapsed', expanded);
          summaryToggle.textContent = expanded ? 'Show more' : 'Show less';
        });
      }
    }
  };

  const enterEditMode = folderId => {
    if (!mdOriginalMetadata) return;
    mdEditMode = true;
    mdHeaderActions.style.display = 'none';
    communityScoreWidget.style.display = 'none';
    const md = mdOriginalMetadata;
    const locks = md.locks || {};

    pageTitleEl.innerHTML =
      `<input type="text" class="md-edit-input md-edit-title" data-field="title" value="${escapeHtml(md.title || '')}" placeholder="Title">` +
      lockIconHtml(locks.title, 'title_lock');

    let statusOpts = `<option value=""${!md.status ? ' selected' : ''}>— No status —</option>`;
    ['ONGOING', 'COMPLETED', 'ABANDONED', 'HIATUS'].forEach(s => {
      statusOpts += `<option value="${s}"${md.status === s ? ' selected' : ''}>${s}</option>`;
    });
    mdStatusBadge.className = 'md-status-badge';
    mdStatusBadge.innerHTML =
      `<select class="md-edit-select md-edit-status" data-field="status">${statusOpts}</select>` +
      lockIconHtml(locks.status, 'status_lock');
    mdStatusBadge.style.display = 'inline-flex';

    const html = buildMetadataPanelHtml(md, locks, mdOriginalProvider, true);
    metadataPanel.innerHTML = html;
    metadataPanel.style.display = 'block';
    attachPanelListeners(folderId, true);
  };

  const handleMetadataCancel = () => {
    mdEditMode = false;
    if (!mdOriginalMetadata || !state.currentFolderId) return;
    const md = mdOriginalMetadata;
    const locks = md.locks || {};
    renderMetadataInHeader(md, locks, mdOriginalProvider);
    const html = buildMetadataPanelHtml(md, locks, mdOriginalProvider, false);
    metadataPanel.innerHTML = html;
    metadataPanel.style.display = html.trim() ? 'block' : 'none';
    attachPanelListeners(state.currentFolderId, false);
  };

  const handleMetadataSave = async folderId => {
    if (!mdOriginalMetadata) return;
    const orig = mdOriginalMetadata;

    const getVal = field => {
      const el =
        metadataPanel.querySelector(`[data-field="${field}"]`) ||
        document.querySelector(`.page-header-left [data-field="${field}"]`);
      if (!el) return undefined;
      return el.value;
    };

    const numericFields = [
      'age_rating',
      'community_score',
      'release_year',
      'release_month',
      'release_day',
      'total_book_count',
    ];
    let hasErrors = false;
    numericFields.forEach(field => {
      const el = metadataPanel.querySelector(`[data-field="${field}"]`);
      if (!el) return;
      const err = validateNumericField(el.value, field);
      if (err) {
        showFieldError(el, err);
        hasErrors = true;
      } else {
        clearFieldError(el);
      }
    });
    if (hasErrors) return;

    // Build PATCH body with only changed scalar fields
    const body = {};

    ['title', 'summary', 'publisher', 'language'].forEach(field => {
      const val = getVal(field);
      if (val === undefined) return;
      const origVal = orig[field] || '';
      if (val !== origVal) {
        body[field] = val === '' ? null : val;
      }
    });

    ['status', 'reading_direction'].forEach(field => {
      const val = getVal(field);
      if (val === undefined) return;
      const origVal = orig[field] || '';
      if (val !== origVal) {
        body[field] = val === '' ? null : val;
      }
    });

    numericFields.forEach(field => {
      const val = getVal(field);
      if (val === undefined) return;
      const origVal = orig[field] !== null && orig[field] !== undefined ? String(orig[field]) : '';
      if (val !== origVal) {
        if (val === '') {
          body[field] = null;
        } else {
          body[field] = field === 'community_score' ? parseFloat(val) : parseInt(val, 10);
        }
      }
    });

    if (Object.keys(body).length === 0) {
      toast.success('No changes to save');
      mdEditMode = false;
      fetchAndRenderMetadataPanel(folderId);
      return;
    }

    const saveBtn = metadataPanel.querySelector('#md-save-btn');
    if (saveBtn) {
      saveBtn.disabled = true;
      saveBtn.innerHTML =
        '<div class="md-spinner" style="width:12px;height:12px;border-width:2px;"></div> Saving...';
    }

    try {
      const res = await fetch(`/api/folders/${folderId}/metadata`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });

      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        throw new Error(errData.error || `Save failed (HTTP ${res.status})`);
      }

      toast.success('Metadata saved');
      mdEditMode = false;
      fetchAndRenderMetadataPanel(folderId);
    } catch (err) {
      toast.error(err.message);
      // Stay in edit mode, re-enable save button
      if (saveBtn) {
        saveBtn.disabled = false;
        saveBtn.innerHTML = '<i class="ph-bold ph-floppy-disk"></i> Save';
      }
    }
  };

  const renderMetadataInHeader = (md, locks, provider) => {
    pageTitleEl.style.display = '';
    if (md.title && md.title.trim()) {
      pageTitleEl.innerHTML = escapeHtml(md.title) + lockIconHtml(locks.title, 'title_lock');
    } else {
      pageTitleEl.textContent = originalFolderName;
    }

    if (md.status) {
      const statusClass = md.status.toLowerCase();
      mdStatusBadge.className = `md-status-badge ${statusClass}`;
      mdStatusBadge.innerHTML = escapeHtml(md.status) + lockIconHtml(locks.status, 'status_lock');
      mdStatusBadge.style.display = 'inline-flex';
    } else {
      mdStatusBadge.style.display = 'none';
    }

    if (md.community_score !== null && md.community_score !== undefined) {
      csValue.innerHTML =
        `${md.community_score.toFixed(1)}/10` +
        lockIconHtml(locks.community_score, 'community_score_lock');
      communityScoreWidget.style.display = 'flex';
    } else {
      communityScoreWidget.style.display = 'none';
    }

    mdHeaderActions.style.display = provider ? 'inline-flex' : 'none';
  };

  const clearMetadataFromHeader = () => {
    pageTitleEl.style.display = '';
    pageTitleEl.textContent = originalFolderName;
    mdStatusBadge.style.display = 'none';
    communityScoreWidget.style.display = 'none';
    mdHeaderActions.style.display = 'none';
  };

  const refreshFolderTags = async folderId => {
    try {
      const params = new URLSearchParams({
        folderId,
        page: 1,
        per_page: 1,
        sort_by: 'auto',
        sort_dir: 'asc',
      });
      const res = await fetch(`/api/browse?${params}`);
      if (res.ok) {
        const data = await res.json();
        if (data.current_folder) {
          currentFolderTags = data.current_folder.tags || [];
          renderTags(currentFolderTags, true);
        }
      }
    } catch (e) {
      console.warn('Failed to refresh folder tags:', e);
    }
  };

  const hasMetadataContent = md => {
    return !!(
      md.title ||
      md.status ||
      (md.summary && md.summary.trim()) ||
      md.publisher ||
      md.reading_direction ||
      md.language ||
      (md.age_rating !== null && md.age_rating !== undefined) ||
      (md.total_book_count !== null && md.total_book_count !== undefined) ||
      (md.release_year !== null && md.release_year !== undefined) ||
      (md.community_score !== null && md.community_score !== undefined) ||
      (md.titles && md.titles.length > 0) ||
      (md.authors && md.authors.length > 0) ||
      (md.links && md.links.length > 0)
    );
  };

  const fetchAndRenderMetadataPanel = async folderId => {
    if (!folderId || !metadataPanel || !noMetadataPrompt) return;

    try {
      const res = await fetch(`/api/folders/${folderId}/metadata`);

      if (res.status === 404) {
        metadataPanel.style.display = 'none';
        metadataPanel.innerHTML = '';
        noMetadataPrompt.style.display = 'block';
        clearMetadataFromHeader();
        mdOriginalMetadata = null;
        mdOriginalProvider = null;
        mdEditMode = false;
        return;
      }

      if (!res.ok) {
        throw new Error(`HTTP ${res.status}`);
      }

      const data = await res.json();
      const md = data.metadata;
      const locks = md.locks || {};
      const provider = data.provider;

      mdOriginalMetadata = md;
      mdOriginalProvider = provider;
      mdEditMode = false;

      noMetadataPrompt.style.display = 'none';

      renderMetadataInHeader(md, locks, provider);

      const html = buildMetadataPanelHtml(md, locks, provider, false);

      if (!hasMetadataContent(md) && !provider) {
        metadataPanel.style.display = 'none';
        metadataPanel.innerHTML = '';
        noMetadataPrompt.style.display = 'block';
        return;
      }

      metadataPanel.innerHTML = html;
      metadataPanel.style.display = html.trim() ? 'block' : 'none';

      attachLockListeners(folderId);
      attachPanelListeners(folderId, false);
    } catch (error) {
      console.error('Error fetching metadata:', error);
      toast.error('Failed to load metadata');
    }
  };

  const openMetadataSearchModal = () => {
    mdSelectedResult = null;
    mdLinkBtn.disabled = true;
    mdSearchResults.innerHTML = '';
    mdSearchError.style.display = 'none';
    mdSearchLoading.style.display = 'none';
    mdSearchInput.value = originalFolderName || pageTitleEl.textContent || '';
    metadataSearchModal.style.display = 'flex';
    mdSearchInput.focus();
    mdSearchInput.select();
  };

  const closeMetadataSearchModal = () => {
    metadataSearchModal.style.display = 'none';
    mdSelectedResult = null;
  };

  const performMetadataSearch = async () => {
    const query = mdSearchInput.value.trim();
    if (!query) return;

    mdSearchError.style.display = 'none';
    mdSearchResults.innerHTML = '';
    mdSearchLoading.style.display = 'flex';
    mdSelectedResult = null;
    mdLinkBtn.disabled = true;

    try {
      const res = await fetch(`/api/metadata/search?q=${encodeURIComponent(query)}&limit=10`);
      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        throw new Error(errData.error || `Search failed (HTTP ${res.status})`);
      }
      const data = await res.json();
      const results = data.results || [];

      mdSearchLoading.style.display = 'none';

      if (results.length === 0) {
        mdSearchResults.innerHTML =
          '<div class="md-search-empty">No results found. Try a different search query.</div>';
        return;
      }

      mdSearchResults.innerHTML = results
        .map(
          (r, i) => `<div class="md-result-card" data-index="${i}">
          <img class="md-result-thumb" src="${escapeHtml(r.image_url || '')}" alt="" onerror="this.style.display='none'">
          <div class="md-result-info">
            <div class="md-result-title">${escapeHtml(r.title)}</div>
            <div class="md-result-provider">${escapeHtml(r.provider_name)}</div>
          </div>
        </div>`
        )
        .join('');

      mdSearchResults.querySelectorAll('.md-result-card').forEach((card, idx) => {
        card.addEventListener('click', () => {
          mdSearchResults
            .querySelectorAll('.md-result-card')
            .forEach(c => c.classList.remove('selected'));
          card.classList.add('selected');
          mdSelectedResult = results[idx];
          mdLinkBtn.disabled = false;
        });
      });
    } catch (err) {
      mdSearchLoading.style.display = 'none';
      mdSearchError.textContent = err.message;
      mdSearchError.style.display = 'block';
    }
  };

  const handleMetadataLink = async () => {
    if (!mdSelectedResult || !state.currentFolderId) return;

    mdLinkBtn.disabled = true;
    mdLinkBtn.innerHTML =
      '<div class="md-spinner" style="width:14px;height:14px;border-width:2px;"></div> Linking...';
    mdSearchError.style.display = 'none';

    try {
      const res = await fetch(`/api/folders/${state.currentFolderId}/metadata/link`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          provider_name: mdSelectedResult.provider_name,
          provider_id: mdSelectedResult.result_id,
        }),
      });

      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        throw new Error(errData.error || `Link failed (HTTP ${res.status})`);
      }

      closeMetadataSearchModal();
      toast.success('Metadata linked successfully');
      fetchAndRenderMetadataPanel(state.currentFolderId);
      refreshFolderTags(state.currentFolderId);
    } catch (err) {
      mdSearchError.textContent = err.message;
      mdSearchError.style.display = 'block';
    } finally {
      mdLinkBtn.innerHTML = '<i class="ph-bold ph-link"></i> Link';
      mdLinkBtn.disabled = !mdSelectedResult;
    }
  };

  const handleMetadataRefresh = async folderId => {
    mdRefreshBtn.disabled = true;

    try {
      const res = await fetch(`/api/folders/${folderId}/metadata/refresh`, {
        method: 'POST',
      });
      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        throw new Error(errData.error || `Refresh failed (HTTP ${res.status})`);
      }
      toast.success('Metadata refreshed');
      fetchAndRenderMetadataPanel(folderId);
      refreshFolderTags(folderId);
    } catch (err) {
      toast.error(err.message);
    } finally {
      mdRefreshBtn.disabled = false;
    }
  };

  const handleMetadataReset = async folderId => {
    if (!confirm('This will clear all metadata but keep the provider link. Continue?')) return;

    try {
      const res = await fetch(`/api/folders/${folderId}/metadata/reset`, {
        method: 'POST',
      });
      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        throw new Error(errData.error || `Reset failed (HTTP ${res.status})`);
      }
      toast.success('Metadata reset');
      fetchAndRenderMetadataPanel(folderId);
      refreshFolderTags(folderId);
    } catch (err) {
      toast.error(err.message);
    }
  };

  const handleMetadataUnlink = async folderId => {
    if (!confirm('This will remove all metadata AND the provider link. Continue?')) return;

    try {
      const res = await fetch(`/api/folders/${folderId}/metadata/unlink`, {
        method: 'POST',
      });
      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        throw new Error(errData.error || `Unlink failed (HTTP ${res.status})`);
      }
      toast.success('Metadata unlinked');
      fetchAndRenderMetadataPanel(folderId);
      refreshFolderTags(folderId);
    } catch (err) {
      toast.error(err.message);
    }
  };

  // Get the current folder ID from the URL path.
  const getFolderIdFromUrl = () => {
    const parts = window.location.pathname.split('/folder/');
    return parts.length > 1 ? parts[1] : null;
  };
  const getTagIdFromUrl = () => {
    const parts = window.location.pathname.split('/tags/');
    return parts.length > 1 ? parts[1] : null;
  };
  const getTagNameFromId = async id => {
    const tag = allTags.find(tag => tag.id === parseInt(id));
    return tag ? tag.name : '';
  };

  // Renders the horizontal tag filter bar shown only at the library root.
  // Hidden when inside a folder or browsing by a URL tag.
  const renderTagFilterBar = () => {
    const isRoot = !state.currentFolderId && !state.currentTagId;
    if (!isRoot || allTags.length === 0) {
      filterChipsExpanded = false;
      tagFilterBar.style.display = 'none';
      return;
    }
    tagFilterBar.style.display = 'flex';
    tagFilterChips.innerHTML = '';

    // null sentinel represents the "All" chip (clears the tag filter)
    const allItems = [null, ...allTags];
    const overflow = allItems.length > FILTER_CHIPS_LIMIT;

    let toShow =
      !filterChipsExpanded && overflow ? allItems.slice(0, FILTER_CHIPS_LIMIT) : allItems;

    // Always keep the active tag visible even if it falls beyond the limit,
    // so the user can see which filter is applied without having to expand first.
    if (!filterChipsExpanded && overflow && state.filterTagId !== null) {
      const activeIdx = allItems.findIndex(t => t !== null && t.id === state.filterTagId);
      if (activeIdx >= FILTER_CHIPS_LIMIT) {
        toShow = [...allItems.slice(0, FILTER_CHIPS_LIMIT - 1), allItems[activeIdx]];
      }
    }

    // .expanded switches the row from nowrap-scroll to wrapping layout
    tagFilterChips.classList.toggle('expanded', filterChipsExpanded || !overflow);

    toShow.forEach(tag => {
      const chip = document.createElement('button');
      if (tag === null) {
        chip.className = 'tag-chip' + (state.filterTagId === null ? ' active' : '');
        chip.textContent = 'All';
        chip.addEventListener('click', () => {
          state.filterTagId = null;
          state.currentPage = 1;
          renderTagFilterBar();
          loadFolderContents();
        });
      } else {
        chip.className = 'tag-chip' + (state.filterTagId === tag.id ? ' active' : '');
        chip.textContent = tag.name;
        chip.addEventListener('click', () => {
          // Clicking the active chip again clears the filter (toggle behaviour)
          state.filterTagId = state.filterTagId === tag.id ? null : tag.id;
          state.currentPage = 1;
          renderTagFilterBar();
          loadFolderContents();
        });
      }
      tagFilterChips.appendChild(chip);
    });

    if (overflow) {
      const hidden = allItems.length - toShow.length;
      const toggleEl = document.createElement('button');
      toggleEl.className = 'tag-show-more';
      if (!filterChipsExpanded) {
        toggleEl.textContent = hidden > 0 ? `+${hidden} more` : 'show less';
        if (hidden > 0) {
          toggleEl.addEventListener('click', () => {
            filterChipsExpanded = true;
            renderTagFilterBar();
          });
        } else {
          toggleEl.addEventListener('click', () => {
            filterChipsExpanded = false;
            renderTagFilterBar();
          });
        }
      } else {
        toggleEl.textContent = 'show less';
        toggleEl.addEventListener('click', () => {
          filterChipsExpanded = false;
          renderTagFilterBar();
        });
      }
      tagFilterChips.appendChild(toggleEl);
    }
  };

  // Renders the 1-10 rating stars for the current folder.
  const renderRating = rating => {
    state.currentRating = rating ?? null;
    ratingStars.innerHTML = '';
    for (let i = 1; i <= 5; i++) {
      const star = document.createElement('button');
      star.className = 'rating-star' + (i <= (rating ?? 0) ? ' filled' : '');
      star.textContent = '★';
      star.dataset.value = i;
      star.addEventListener('click', () => setRating(i));
      star.addEventListener('mouseover', () => highlightStars(i));
      star.addEventListener('mouseout', () => highlightStars(state.currentRating ?? 0));
      ratingStars.appendChild(star);
    }
  };

  const highlightStars = value => {
    ratingStars.querySelectorAll('.rating-star').forEach((s, idx) => {
      s.classList.toggle('filled', idx < value);
    });
  };

  const setRating = async value => {
    if (!state.currentFolderId) return;
    const body = { rating: state.currentRating === value ? 0 : value };
    const res = await fetch(`/api/folders/${state.currentFolderId}/rating`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (res.ok) {
      const data = await res.json();
      renderRating(data.rating);
    }
  };

  // Fetches and renders the breadcrumb navigation.
  const renderBreadcrumb = async () => {
    if (!state.currentFolderId) {
      breadcrumbEl.innerHTML = '';
      return;
    }
    if (breadcrumbEl.innerHTML != '') {
      return;
    }
    const url = `/api/browse/breadcrumb?folderId=${state.currentFolderId}`;
    const response = await fetch(url);
    const path = await response.json();

    let html = '<a href="/library">Library</a>';
    path.forEach(folder => {
      html += ` / <a href="/library/folder/${folder.id}">${folder.name}</a>`;
    });
    breadcrumbEl.innerHTML = html;
  };

  // Renders the grid with folders first, then chapters.
  const renderGrid = data => {
    cardsGrid.innerHTML = '';
    if (
      (!data.subfolders || data.subfolders.length === 0) &&
      (!data.chapters || data.chapters.length === 0)
    ) {
      const hasActiveFilter =
        state.search || state.unreadOnly || state.filterTagId || state.currentTagId;
      cardsGrid.innerHTML = hasActiveFilter
        ? '<p>No results found.</p>'
        : '<p>This folder is empty.</p>';
      return;
    }

    if (data.subfolders && data.subfolders.length > 0) {
      cardsGrid.insertAdjacentHTML('beforeend', '<h3 class="grid-section-header">Folders</h3>');
      data.subfolders.forEach(folder => cardsGrid.appendChild(createFolderCard(folder)));
    }
    if (data.chapters && data.chapters.length > 0) {
      cardsGrid.insertAdjacentHTML('beforeend', '<h3 class="grid-section-header">Chapters</h3>');
      data.chapters.forEach(chapter => cardsGrid.appendChild(createChapterCard(chapter)));
    }

    // Load AniList buttons asynchronously after rendering
    loadAniListButtonsAsync();
  };

  // Creates an HTML card for a folder.
  const createFolderCard = folder => {
    const card = document.createElement('a');
    card.href = `/library/folder/${folder.id}`;
    card.className = 'item-card folder';
    // Calculate progress for the folder
    const progressPercent =
      folder.total_chapters > 0 ? (folder.read_chapters / folder.total_chapters) * 100 : 0;
    const ratingBadge = folder.rating ? `<div class="rating-badge">★ ${folder.rating}</div>` : '';
    card.innerHTML = `
            <div class="thumbnail-container">
                <img class="thumbnail" src="${folder.thumbnail || '/static/images/logo.svg'}" loading="lazy" alt="Cover for ${folder.name}">
                ${ratingBadge}
            </div>
            <div class="item-title" title="${folder.name}">${folder.name}</div>
            <div class="progress-bar-container">
              <div class="progress-bar" style="width: ${progressPercent}%;"></div>
            </div>
        `;
    return card;
  };

  // Creates an HTML card for a chapter.
  const createChapterCard = chapter => {
    const card = document.createElement('a');
    const progressPercent = chapter.progress_percent || 0;
    card.href = `/reader/series/${chapter.folder_id}/chapters/${chapter.id}`; // Note: Reader URL might need adjustment
    card.className = 'item-card';
    const title = chapter.path.split(/[\\\\/]/).pop();
    card.innerHTML = `
            <div class="thumbnail-container">
                <img class="thumbnail" src="${chapter.thumbnail || ''}" loading="lazy" alt="Cover for ${title}">
            </div>
            <div class="item-title" title="${title}">${title}</div>
            <div class="progress-bar-container">
                <div class="progress-bar" style="width: ${progressPercent}%;"></div>
            </div>
        `;
    return card;
  };

  const saveFolderSettings = async () => {
    await fetch(`/api/folders/${state.currentFolderId}/settings`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ sort_by: state.sortBy, sort_dir: state.sortDir }),
    });
  };

  // Fetch folder settings and update state
  const loadFolderSettings = async () => {
    if (!state.currentFolderId) return;
    if (state.sortBy != null && state.sortDir != null) return;

    try {
      const response = await fetch(`/api/folders/${state.currentFolderId}/settings`);
      if (response.ok) {
        const settings = await response.json();
        state.sortBy = settings.sort_by || 'auto';
        state.sortDir = settings.sort_dir || 'asc';

        // Update UI elements to reflect the loaded settings
        sortBySelect.value = state.sortBy;
        sortDirBtn.textContent = state.sortDir === 'asc' ? '▲' : '▼';
      }
    } catch (error) {
      console.error('Failed to load folder settings:', error);
    }
  };

  // Main function to fetch all data and render the page.
  const loadFolderContents = async () => {
    if (state.isLoading) return;
    state.isLoading = true;
    cardsGrid.innerHTML = '<p>Loading...</p>';

    try {
      await renderBreadcrumb();
      await loadFolderSettings();

      const params = new URLSearchParams({
        page: state.currentPage,
        per_page: state.perPage,
        search: state.search,
        sort_by: state.sortBy,
        sort_dir: state.sortDir,
      });
      if (state.currentFolderId) {
        params.set('folderId', state.currentFolderId);
      }
      const activeTagId = state.currentTagId || state.filterTagId;
      if (activeTagId) {
        params.set('tagId', activeTagId);
      }
      if (state.unreadOnly) {
        params.set('unread_only', 'true');
      }

      const response = await fetch(`/api/browse?${params.toString()}`);

      if (!response.ok) {
        console.error(`Browse API error: ${response.status} ${response.statusText}`);
        throw new Error(`Browse API error: ${response.status}`);
      }

      const data = await response.json();

      if (data.current_folder) {
        originalFolderName = data.current_folder.name;
        pageTitleEl.textContent = data.current_folder.name;
      } else if (state.currentTagId) {
        originalFolderName = '';
        const tagName = await getTagNameFromId(state.currentTagId);
        pageTitleEl.textContent = `Tag: ${tagName}`;
        document.title = `Tag: ${tagName} - Mango`;
      } else {
        originalFolderName = '';
        pageTitleEl.textContent = 'Library';
      }
      document.title = `${pageTitleEl.textContent} - Mango`;
      folderThumb.src = data.current_folder ? data.current_folder.thumbnail : '';
      folderThumb.style.display = data.current_folder ? 'block' : 'none';

      const inFolder = !!data.current_folder;

      editFolderBtn.style.display = inFolder ? 'block' : 'none';
      progressActions.style.display = inFolder ? 'flex' : 'none';
      folderTagsSection.style.display = inFolder ? 'flex' : 'none';
      renderTags(inFolder ? data.current_folder.tags : []);

      if (inFolder) {
        ratingWidget.style.display = 'flex';
        renderRating(data.current_folder.rating ?? null);
      } else {
        ratingWidget.style.display = 'none';
      }

      if (inFolder) {
        fetchAndRenderMetadataPanel(state.currentFolderId);
      } else {
        metadataPanel.style.display = 'none';
        metadataPanel.innerHTML = '';
        noMetadataPrompt.style.display = 'none';
        clearMetadataFromHeader();
      }

      // Tag filter chips at root level
      renderTagFilterBar();

      // Unread button active state
      unreadFilterBtn.classList.toggle('active', state.unreadOnly);

      renderGrid(data);

      state.totalItems = parseInt(response.headers.get('X-Total-Count') || '0', 10);
      totalCountEl.textContent = `${state.totalItems}`;
      renderPagination();
    } catch (error) {
      console.error('Error loading folder contents:', error);
      cardsGrid.innerHTML = '<p>Error loading content. Please try again.</p>';
    } finally {
      state.isLoading = false;
    }
  };

  const renderPagination = () => {
    paginationContainer.innerHTML = '';
    const totalPages = Math.ceil(state.totalItems / state.perPage);
    if (totalPages <= 1) return;

    const createButton = (text, page, isDisabled = false, isActive = false) => {
      const btn = document.createElement('button');
      btn.className = 'pagination-btn';
      btn.innerHTML = text;
      if (isDisabled) btn.classList.add('disabled');
      if (isActive) btn.classList.add('active');
      btn.addEventListener('click', () => {
        state.currentPage = page;
        loadFolderContents();
        btn.classList.add('active');
        const siblings = Array.from(paginationContainer.children);
        siblings.forEach(sibling => {
          if (sibling !== btn) sibling.classList.remove('active');
        });
      });
      return btn;
    };

    paginationContainer.appendChild(createButton('&laquo;', 1, state.currentPage === 1));
    paginationContainer.appendChild(
      createButton('&lsaquo;', state.currentPage - 1, state.currentPage === 1)
    );

    const pageNumbers = [];
    // Always show first page
    pageNumbers.push(1);

    // Ellipsis logic
    if (state.currentPage > 4) {
      pageNumbers.push('...');
    }

    // Window of pages around current page
    for (
      let i = Math.max(2, state.currentPage - 2);
      i <= Math.min(totalPages - 1, state.currentPage + 2);
      i++
    ) {
      pageNumbers.push(i);
    }

    if (state.currentPage < totalPages - 3) {
      pageNumbers.push('...');
    }

    // Always show last page
    if (totalPages > 1) pageNumbers.push(totalPages);

    // Render buttons from unique page numbers
    [...new Set(pageNumbers)].forEach(num => {
      if (num === '...') {
        const ellipsis = document.createElement('span');
        ellipsis.className = 'pagination-ellipsis';
        ellipsis.textContent = '...';
        paginationContainer.appendChild(ellipsis);
      } else {
        paginationContainer.appendChild(createButton(num, num, false, state.currentPage === num));
      }
    });

    paginationContainer.appendChild(
      createButton('&rsaquo;', state.currentPage + 1, state.currentPage === totalPages)
    );
    paginationContainer.appendChild(
      createButton('&raquo;', totalPages, state.currentPage === totalPages)
    );
  };

  // --- Tagging Logic ---
  const loadAllTags = async () => {
    try {
      const response = await fetch('/api/tags');
      allTags = (await response.json()) || [];
    } catch (e) {
      console.error('Failed to load all tags:', e);
    }
  };

  // Renders inline folder tags with collapse/expand behaviour.
  // preserveExpanded=true keeps the current expand state (used after add/remove/toggle);
  // false (default) resets to collapsed when navigating to a new folder.
  const renderTags = (tags, preserveExpanded = false) => {
    if (!preserveExpanded) tagsExpanded = false;
    currentFolderTags = tags || [];
    tagsContainer.innerHTML = '';

    const overflow = currentFolderTags.length > TAGS_COLLAPSED_LIMIT;
    const toShow =
      !tagsExpanded && overflow
        ? currentFolderTags.slice(0, TAGS_COLLAPSED_LIMIT)
        : currentFolderTags;

    // .expanded switches from nowrap-scroll to wrapping layout
    tagsContainer.classList.toggle('expanded', tagsExpanded || !overflow);

    toShow.forEach(tag => {
      const tagEl = document.createElement('div');
      tagEl.className = 'tag';

      const nameSpan = document.createElement('span');
      nameSpan.className = 'tag-name';
      nameSpan.textContent = tag.name;

      const removeBtn = document.createElement('button');
      removeBtn.className = 'tag-remove-btn';
      removeBtn.title = `Remove ${tag.name}`;
      removeBtn.textContent = '×';
      // Direct listener (not delegation) so the hit-target is reliable
      removeBtn.addEventListener('click', e => {
        e.stopPropagation();
        removeTag(tag.id);
      });

      tagEl.appendChild(nameSpan);
      tagEl.appendChild(removeBtn);
      tagsContainer.appendChild(tagEl);
    });

    if (overflow) {
      const toggleEl = document.createElement('button');
      toggleEl.className = 'tag-show-more';
      if (!tagsExpanded) {
        toggleEl.textContent = `+${currentFolderTags.length - TAGS_COLLAPSED_LIMIT} more`;
        toggleEl.addEventListener('click', () => {
          tagsExpanded = true;
          renderTags(currentFolderTags, true);
        });
      } else {
        toggleEl.textContent = 'show less';
        toggleEl.addEventListener('click', () => {
          tagsExpanded = false;
          renderTags(currentFolderTags, true);
        });
      }
      tagsContainer.appendChild(toggleEl);
    }
  };

  const addTag = async tagName => {
    const normalizedTagName = tagName.trim().toLowerCase();
    if (
      normalizedTagName === '' ||
      !state.currentFolderId ||
      currentFolderTags.some(t => t.name === normalizedTagName)
    ) {
      tagInput.value = '';
      return;
    }
    const response = await fetch(`/api/folders/${state.currentFolderId}/tags`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: normalizedTagName }),
    });
    if (response.ok) {
      tagInput.value = '';
      autocompleteSuggestions.style.display = 'none';
      const newTag = await response.json();
      currentFolderTags.push(newTag);
      renderTags(currentFolderTags, true);
    }
  };

  const removeTag = async tagId => {
    await fetch(`/api/folders/${state.currentFolderId}/tags/${tagId}`, {
      method: 'DELETE',
    });
    currentFolderTags = currentFolderTags.filter(t => t.id != tagId);
    renderTags(currentFolderTags, true);
  };

  const handleSearch = () => {
    state.search = searchInput.value.trim();
    state.currentPage = 1;
    loadFolderContents();
  };

  const handleSaveChanges = async () => {
    const file = coverFileInput.files[0];
    if (!file) {
      // In the future, you could handle other fields here.
      // For now, if no file, just close the modal.
      editFolderModal.style.display = 'none';
      return;
    }

    const formData = new FormData();
    formData.append('cover_file', file);

    try {
      const response = await fetch(`/api/folders/${state.currentFolderId}/cover`, {
        method: 'POST',
        body: formData, // The browser will set the Content-Type header automatically
      });

      if (response.ok) {
        editFolderModal.style.display = 'none';
        toast.success('Cover uploaded successfully!');
        loadFolderContents(); // Reload to show the new cover
      } else {
        const errorData = await response.json();
        toast.error(`Error uploading cover: ${errorData.error}`);
      }
    } catch (err) {
      toast.error('An unexpected error occurred during upload.');
    }
  };

  // Mark all chapters as read or unread.
  const markAllAs = async read => {
    const response = await fetch(`/api/folders/${state.currentFolderId}/mark-all-as`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ read }),
    });
    if (response.ok) {
      toast.success(`All chapters marked as ${read ? 'read' : 'unread'}`);
      loadFolderContents();
    } else {
      toast.error(`Failed to mark all chapters as ${read ? 'read' : 'unread'}`);
    }
  };
  // --- Event Listeners ---
  editFolderBtn.addEventListener('click', () => {
    if (state.currentFolderId) {
      editFolderModal.style.display = 'flex';
      coverFileInput.value = ''; // Clear previous selection
      selectedFileInfo.style.display = 'none'; // Hide file info
      // Update save button text to be more intuitive
      modalSaveBtn.innerHTML = '<i class="ph-bold ph-upload"></i> Upload Cover';
    }
  });

  // Handle file selection
  coverFileInput.addEventListener('change', e => {
    const file = e.target.files[0];
    if (file) {
      fileName.textContent = file.name;
      selectedFileInfo.style.display = 'flex';
    } else {
      selectedFileInfo.style.display = 'none';
    }
  });

  // Handle remove file button
  removeFileBtn.addEventListener('click', () => {
    coverFileInput.value = '';
    selectedFileInfo.style.display = 'none';
  });
  modalCancelBtn.addEventListener('click', () => (editFolderModal.style.display = 'none'));
  modalSaveBtn.addEventListener('click', handleSaveChanges);
  modalCloseBtn.addEventListener('click', () => (editFolderModal.style.display = 'none'));

  // Close modal when clicking outside
  editFolderModal.addEventListener('click', e => {
    if (e.target === editFolderModal) {
      editFolderModal.style.display = 'none';
    }
  });

  // Close modal when pressing ESC key
  document.addEventListener('keydown', e => {
    if (e.key === 'Escape' && editFolderModal.style.display === 'flex') {
      editFolderModal.style.display = 'none';
    }
    if (e.key === 'Escape' && metadataSearchModal.style.display === 'flex') {
      closeMetadataSearchModal();
    }
  });

  linkMetadataBtn.addEventListener('click', openMetadataSearchModal);

  mdEditBtn.addEventListener('click', () => {
    if (state.currentFolderId) enterEditMode(state.currentFolderId);
  });
  mdRefreshBtn.addEventListener('click', () => {
    if (state.currentFolderId) handleMetadataRefresh(state.currentFolderId);
  });
  mdRelinkBtn.addEventListener('click', openMetadataSearchModal);
  mdResetBtn.addEventListener('click', () => {
    if (state.currentFolderId) handleMetadataReset(state.currentFolderId);
  });
  mdUnlinkBtn.addEventListener('click', () => {
    if (state.currentFolderId) handleMetadataUnlink(state.currentFolderId);
  });

  mdSearchBtn.addEventListener('click', performMetadataSearch);
  mdSearchInput.addEventListener('keydown', e => {
    if (e.key === 'Enter') {
      e.preventDefault();
      performMetadataSearch();
    }
  });
  mdLinkBtn.addEventListener('click', handleMetadataLink);
  mdSearchCloseBtn.addEventListener('click', closeMetadataSearchModal);
  mdSearchCancelBtn.addEventListener('click', closeMetadataSearchModal);
  metadataSearchModal.addEventListener('click', e => {
    if (e.target === metadataSearchModal) {
      closeMetadataSearchModal();
    }
  });

  tagInput.addEventListener('keydown', e => {
    if (e.key === 'Enter') {
      e.preventDefault();
      addTag(tagInput.value);
    }
  });
  tagInput.addEventListener('input', () => {
    const query = tagInput.value.trim().toLowerCase();
    if (query === '') {
      autocompleteSuggestions.style.display = 'none';
      return;
    }
    const suggestions = allTags.filter(
      tag =>
        tag.name.toLowerCase().includes(query) && !currentFolderTags.some(t => t.name === tag.name)
    );
    autocompleteSuggestions.innerHTML = '';
    suggestions.slice(0, 5).forEach(suggestion => {
      const suggestionEl = document.createElement('div');
      suggestionEl.className = 'autocomplete-suggestion';
      suggestionEl.textContent = suggestion.name;
      suggestionEl.addEventListener('click', () => {
        addTag(suggestion.name);
        autocompleteSuggestions.style.display = 'none';
      });
      autocompleteSuggestions.appendChild(suggestionEl);
    });
    autocompleteSuggestions.style.display = suggestions.length > 0 ? 'block' : 'none';
  });

  let searchTimeout;
  searchInput.addEventListener('input', () => {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(handleSearch, 300);
  });

  sortBySelect.addEventListener('change', () => {
    state.sortBy = sortBySelect.value;
    state.sortDir = state.sortDir === 'asc' ? 'desc' : 'asc';
    saveFolderSettings();
    loadFolderContents();
  });

  sortDirBtn.addEventListener('click', () => {
    state.sortBy = sortBySelect.value;
    state.sortDir = state.sortDir === 'asc' ? 'desc' : 'asc';
    sortDirBtn.textContent = state.sortDir === 'asc' ? '▲' : '▼';
    saveFolderSettings();
    loadFolderContents();
  });

  markAllReadBtn.addEventListener('click', async () => {
    markAllAs(true);
  });
  markAllUnreadBtn.addEventListener('click', async () => {
    markAllAs(false);
  });

  unreadFilterBtn.addEventListener('click', () => {
    state.unreadOnly = !state.unreadOnly;
    state.currentPage = 1;
    localStorage.setItem('unreadOnly', state.unreadOnly);
    unreadFilterBtn.classList.toggle('active', state.unreadOnly);
    loadFolderContents();
  });

  ratingClearBtn.addEventListener('click', () => setRating(0));

  const init = async () => {
    state.currentFolderId = getFolderIdFromUrl();
    state.currentTagId = getTagIdFromUrl();
    await loadAllTags(); // Load tags for autocomplete
    await loadFolderContents();
  };

  init();
});
