async function search() {
  const qel = document.getElementById('q');
  const q = qel ? qel.value : '';
  const res = await fetch('/search?q=' + encodeURIComponent(q));
  const data = await res.json();
  const out = document.getElementById('results');
  out.innerHTML = '';
  (data.results || []).forEach(p => {
    const el = document.createElement('div');
    el.className = 'pkg';
    el.innerHTML = `<strong>${p.name}</strong> — ${p.description || ''} <button data-id='${p.id}'>View</button>`;
    out.appendChild(el);
    el.querySelector('button').onclick = () => viewPackage(p.id);
  });
}

async function viewPackage(id) {
  const res = await fetch('/packages/' + id);
  const data = await res.json();
  document.getElementById('packageView').style.display = 'block';
  document.getElementById('pkgName').innerText = data.package.name;
  document.getElementById('pkgDesc').innerText = data.package.description || '';

  // load versions
  const vers = await fetch('/packages/' + id + '/versions');
  const vdata = await vers.json();
  const vlist = document.getElementById('versions');
  vlist.innerHTML = '';
  (vdata.versions || []).forEach(v => {
    const li = document.createElement('li');
    li.innerHTML = `${v.version} — <button data-ver='${v.version}'>Select</button>`;
    vlist.appendChild(li);
    li.querySelector('button').onclick = () => selectVersion(id, v.version);
  });

  // attach publish
  const btnPub = document.getElementById('btnPublish');
  if (btnPub) btnPub.onclick = () => publishVersion(id);
}

let selectedVersion = null;
let ACCESS_TOKEN = null;
let CURRENT_USERNAME = null;
let CSRF_TOKEN = null;

async function selectVersion(pkgID, ver) {
  selectedVersion = ver;
  const res = await fetch(`/packages/${pkgID}/versions/${encodeURIComponent(ver)}`);
  const data = await res.json();
  const arts = await fetch(`/packages/${pkgID}/versions/${encodeURIComponent(ver)}/artifacts`);
  const adata = await arts.json();
  const al = document.getElementById('artifacts');
  al.innerHTML = '';
  (adata.artifacts || []).forEach(a => {
    const li = document.createElement('li');
    li.innerHTML = `<a href='/artifacts/${a.id}/download'>${a.filename || a.blob_url}</a> (${a.size_bytes || 0})`;
    al.appendChild(li);
  });
}

async function publishVersion(pkgID) {
  const ver = document.getElementById('newVersion').value;
  const metadata = document.getElementById('newMetadata').value;
  const token = ACCESS_TOKEN;
  const res = await fetch(`/packages/${pkgID}/versions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': token ? 'Bearer ' + token : ''
    },
    body: JSON.stringify({ version: ver, metadata })
  });
  const data = await res.json();
  alert(JSON.stringify(data));
}

async function addArtifact(pkgID) {
  if (!selectedVersion) { alert('select a version'); return }
  const blob = document.getElementById('artifactBlob').value;
  const filename = document.getElementById('artifactFile').value;
  const size = parseInt(document.getElementById('artifactSize').value || '0', 10);
  const token = ACCESS_TOKEN;
  const res = await fetch(`/packages/${pkgID}/versions/${encodeURIComponent(selectedVersion)}/artifacts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Authorization': token ? 'Bearer ' + token : '' },
    body: JSON.stringify({ blob_url: blob, filename, size_bytes: size })
  });
  const data = await res.json();
  alert(JSON.stringify(data));
}

if (document.getElementById('btnSearch')) document.getElementById('btnSearch').onclick = search;
if (document.getElementById('btnAddArtifact')) document.getElementById('btnAddArtifact').onclick = () => {
  const name = document.getElementById('pkgName').innerText;
  fetch('/search?q=' + encodeURIComponent(name)).then(r=>r.json()).then(d=>{
    const p = d.results && d.results[0];
    if (p) addArtifact(p.id);
  });
};

if (document.getElementById('btnComment')) document.getElementById('btnComment').onclick = async () => {
  const body = document.getElementById('commentBody').value;
  const name = document.getElementById('pkgName').innerText;
  const res = await fetch('/search?q=' + encodeURIComponent(name));
  const d = await res.json();
  const p = d.results && d.results[0];
  if (!p) { alert('package not found'); return }
  const r = await fetch(`/packages/${p.id}/comments`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Authorization': ACCESS_TOKEN ? 'Bearer ' + ACCESS_TOKEN : '' },
    body: JSON.stringify({ body })
  });
  alert(await r.text());
};

async function registerUser() {
  const username = document.getElementById('regUsername') ? document.getElementById('regUsername').value : '';
  const email = document.getElementById('regEmail') ? document.getElementById('regEmail').value : '';
  const password = document.getElementById('regPassword') ? document.getElementById('regPassword').value : '';
  const res = await fetch('/register', {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({ username, email, password })
  });
  const data = await res.json();
  const el = document.getElementById('registerResult');
  if (el) el.innerText = JSON.stringify(data);
  const regFields = ['regUsername','regEmail','regPassword'].map(id=>document.getElementById(id)).filter(Boolean);
  if (res.status === 201 || res.status === 200) {
    regFields.forEach(f => f.classList.remove('error-field'));
    const loginRes = await loginWithCredentials(username, password);
    if (loginRes && loginRes.ok && loginRes.data && loginRes.data.token) {
      ACCESS_TOKEN = loginRes.data.token;
      CURRENT_USERNAME = loginRes.data.username || null;
      CSRF_TOKEN = loginRes.data.csrf || null;
      renderAuthState();
      window.location = '/';
    }
  } else {
    regFields.forEach(f => f.classList.add('error-field'));
  }
}

async function loginUser() {
  const username = document.getElementById('loginUsername') ? document.getElementById('loginUsername').value : '';
  const password = document.getElementById('loginPassword') ? document.getElementById('loginPassword').value : '';
  const result = await loginWithCredentials(username, password);
  const loginFields = ['loginUsername','loginPassword'].map(id=>document.getElementById(id)).filter(Boolean);
  if (result && result.ok && result.data && result.data.token) {
    ACCESS_TOKEN = result.data.token;
    CURRENT_USERNAME = result.data.username || null;
    CSRF_TOKEN = result.data.csrf || null;
    loginFields.forEach(f => f.classList.remove('error-field'));
    renderAuthState();
  } else {
    loginFields.forEach(f => f.classList.add('error-field'));
  }
  const regEl = document.getElementById('createPackageResult');
  if (regEl) regEl.innerText = result && result.data ? JSON.stringify(result.data) : 'login failed';
}

async function loginWithCredentials(username, password) {
  if (!username || !password) return null;
  const res = await fetch('/login', {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({ username, password })
  });
  let data = null;
  try { data = await res.json(); } catch (e) { data = null }
  return { ok: res.ok, status: res.status, data };
}

function getCookie(name) {
  const v = `; ${document.cookie}`;
  const parts = v.split(`; ${name}=`);
  if (parts.length === 2) return parts.pop().split(';').shift();
  return null;
}

async function createPackage() {
  const name = document.getElementById('newPkgName') ? document.getElementById('newPkgName').value : '';
  const description = document.getElementById('newPkgDesc') ? document.getElementById('newPkgDesc').value : '';
  const token = ACCESS_TOKEN;
  const res = await fetch('/packages', {
    method: 'POST', headers: { 'Content-Type': 'application/json', 'Authorization': token ? 'Bearer ' + token : '' },
    body: JSON.stringify({ name, description })
  });
  const data = await res.json();
  const el = document.getElementById('createPackageResult'); if (el) el.innerText = JSON.stringify(data);
}

if (document.getElementById('btnRegister')) document.getElementById('btnRegister').onclick = registerUser;
if (document.getElementById('btnLogin')) document.getElementById('btnLogin').onclick = loginUser;
if (document.getElementById('btnCreatePackage')) document.getElementById('btnCreatePackage').onclick = createPackage;

function renderAuthState() {
  const loginForm = document.getElementById('loginForm');
  const userInfo = document.getElementById('userInfo');
  const currentUser = document.getElementById('currentUser');
    if (ACCESS_TOKEN && CURRENT_USERNAME) {
    if (loginForm) loginForm.style.display = 'none';
    if (userInfo) { userInfo.style.display = 'inline-flex'; currentUser.innerText = CURRENT_USERNAME; }
    const btnLogout = document.getElementById('btnLogout');
    if (btnLogout) btnLogout.onclick = async () => {
      try {
        await fetch('/tokens/revoke', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': CSRF_TOKEN || getCookie('ebuild_csrf') || '' } });
      } catch (e) {}
      ACCESS_TOKEN = null;
      CURRENT_USERNAME = null;
      renderAuthState();
      window.location = '/';
    }
    return;
  }
  fetch('/refresh', { method: 'POST', credentials: 'same-origin', headers: { 'X-CSRF-Token': CSRF_TOKEN || getCookie('ebuild_csrf') || '' } }).then(async r => {
    if (!r.ok) {
      if (loginForm) loginForm.style.display = 'block';
      if (userInfo) userInfo.style.display = 'none';
      return;
    }
    const j = await r.json();
    ACCESS_TOKEN = j.token;
    CURRENT_USERNAME = j.username || null;
    CSRF_TOKEN = j.csrf || CSRF_TOKEN;
    if (loginForm) loginForm.style.display = 'none';
    if (userInfo) { userInfo.style.display = 'inline-flex'; currentUser.innerText = CURRENT_USERNAME || '' }
    const btnLogout = document.getElementById('btnLogout');
    if (btnLogout) btnLogout.onclick = async () => {
      try { await fetch('/tokens/revoke', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': CSRF_TOKEN || getCookie('ebuild_csrf') || '' } }); } catch (e) {}
      ACCESS_TOKEN = null; CURRENT_USERNAME = null;
      renderAuthState();
      window.location = '/';
    }
  }).catch(err => {
    if (loginForm) loginForm.style.display = 'block';
    if (userInfo) userInfo.style.display = 'none';
  });
}

renderAuthState();
search();
