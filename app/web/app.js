import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';
import { STLLoader } from 'three/addons/loaders/STLLoader.js';

// --- Internationalization (i18n) Dictionary ---
const translations = {
  en: {
    page_title: 'MoldaAí — 3D Cutter & Mold Generator',
    header_subtitle: 'Transform 2D silhouettes into 3D cookie & clay cutters for 3D printing',
    card1_title: '1. Input Image',
    dropzone_title: 'Click to upload or drag & drop image',
    dropzone_desc: 'PNG, JPEG, WebP or SVG (Solid silhouette or outline)',
    preview_caption: '2D binarized preview (black areas form the cutter)',
    threshold_label: 'Contrast Threshold',
    invert_label: 'Invert Colors (solid ↔ empty)',
    sample_btn: 'Use Sample Cutter (Heart)',
    card2_title: '2. Cutter Dimensions (mm)',
    scale_badge: 'Millimeter Scale',
    preset_small_title: 'Small',
    preset_medium_title: 'Medium',
    preset_large_title: 'Large',
    input_size_label: 'Max Size (mm)',
    input_size_hint: 'Longest axis of cutter',
    input_blade_thick_label: 'Blade Wall (mm)',
    input_blade_thick_hint: 'Ideal 2 perimeters (0.4 nozzle)',
    input_blade_height_label: 'Blade Height (mm)',
    input_blade_height_hint: 'Cutting depth',
    input_base_thick_label: 'Support Rim Width (mm)',
    input_base_thick_hint: 'Comfortable hand grip',
    input_base_height_label: 'Support Rim Height (mm)',
    input_base_height_hint: 'Reinforced base thickness',
    smoothing_label: 'Line Smoothing',
    smoothing_sublabel: '(smoothes wobbles and straightens lines)',
    smoothing_low: '0.1 mm (Fine detail)',
    smoothing_mid: 'Recommended: 0.4 - 0.7 mm',
    smoothing_high: '1.5 mm (Ultra-straight)',
    generate_btn: 'Generate 3D Cutter (.STL)',
    generate_btn_processing: 'Processing Cutter...',
    viewport_title: '3D Viewport',
    viewport_controls: '🖱️ Left click: rotate • Scroll: zoom • Right click: pan',
    empty_title: 'Empty 3D Print Bed',
    empty_desc: 'Upload an image or click the sample cutter to generate the 3D model.',
    loading_text: 'Processing image and building watertight manifold mesh...',
    compatibility_hint: 'Compatible with Cura, PrusaSlicer, Bambu Studio & OrcaSlicer',
    download_btn: 'Download .STL File',
    footer_text: 'MoldaAí © 2026 • Single-Binary Go • Zero Cloud Storage • In-Memory Processing',
    triangles_suffix: 'triangles',
    err_invalid_image: 'Please upload a valid image file (PNG, JPEG, WebP or SVG).',
    err_no_image: 'Please select an image or use the sample cutter.',
    err_height_order: (baseH, bladeH) => `Base height (${baseH} mm) must be less than total blade height (${bladeH} mm).`,
    err_thick_order: (baseT, bladeT) => `Base thickness (${baseT} mm) must be greater than blade thickness (${bladeT} mm).`,
    err_export_blob: 'Failed to export processed image.',
    err_unexpected: 'Unexpected error generating 3D model.',
    success_generated: '3D Cutter (.STL) generated successfully!',
    download_filename: 'cutter-moldaai.stl',
    sample_filename: 'sample-heart.png',
  },
  pt: {
    page_title: 'MoldaAí — Gerador 3D de Cortadores e Moldes',
    header_subtitle: 'Transforme silhuetas 2D em cortadores de biscoito e biscuit para impressão 3D',
    card1_title: '1. Imagem de Entrada',
    dropzone_title: 'Clique para escolher ou arraste a imagem',
    dropzone_desc: 'PNG, JPEG, WebP ou SVG (Silhueta preta ou desenho vazado)',
    preview_caption: 'Preview 2D binarizado da silhueta (o que for preto vira corte)',
    threshold_label: 'Limiar de Contraste (Threshold)',
    invert_label: 'Inverter Cores (sólido ↔ vazio)',
    sample_btn: 'Usar Molde Exemplo (Coração)',
    card2_title: '2. Dimensões do Molde (mm)',
    scale_badge: 'Escala Milimétrica',
    preset_small_title: 'Pequeno',
    preset_medium_title: 'Médio',
    preset_large_title: 'Grande',
    input_size_label: 'Tamanho Máximo (mm)',
    input_size_hint: 'Maior eixo do cortador',
    input_blade_thick_label: 'Parede Lâmina (mm)',
    input_blade_thick_hint: 'Ideal 2 perímetros (bico 0.4)',
    input_blade_height_label: 'Altura da Lâmina (mm)',
    input_blade_height_hint: 'Profundidade do corte',
    input_base_thick_label: 'Aba de Apoio (mm)',
    input_base_thick_hint: 'Pegada confortável de mão',
    input_base_height_label: 'Altura do Apoio (mm)',
    input_base_height_hint: 'Espessura da base reforçada no fundo',
    smoothing_label: 'Amaciamento de Linhas',
    smoothing_sublabel: '(suaviza desvios e alinha retas)',
    smoothing_low: '0.1 mm (Mais detalhes)',
    smoothing_mid: 'Recomendado: 0.4 - 0.7 mm',
    smoothing_high: '1.5 mm (Ultra-reto)',
    generate_btn: 'Gerar Molde 3D (.STL)',
    generate_btn_processing: 'Processando Molde...',
    viewport_title: 'Visualizador 3D',
    viewport_controls: '🖱️ Botão esq: girar • Scroll: zoom • Botão dir: pan',
    empty_title: 'Mesa de Impressão 3D Vazia',
    empty_desc: 'Carregue uma imagem ou clique no molde de exemplo para gerar o cortador 3D.',
    loading_text: 'Processando imagem e construindo malha manifold...',
    compatibility_hint: 'Compatível com Cura, PrusaSlicer, Bambu Studio e OrcaSlicer',
    download_btn: 'Baixar Arquivo .STL',
    footer_text: 'MoldaAí © 2026 • Single-Binary Go • Zero Armazenamento em Nuvem • Processamento In-Memory',
    triangles_suffix: 'triângulos',
    err_invalid_image: 'Por favor envie um arquivo de imagem válido (PNG, JPEG, WebP ou SVG).',
    err_no_image: 'Por favor selecione uma imagem ou use o molde de exemplo.',
    err_height_order: (baseH, bladeH) => `A altura da base (${baseH} mm) deve ser menor que a altura total da lâmina (${bladeH} mm).`,
    err_thick_order: (baseT, bladeT) => `A espessura da base (${baseT} mm) deve ser maior que a espessura da lâmina (${bladeT} mm).`,
    err_export_blob: 'Falha ao exportar imagem processada.',
    err_unexpected: 'Erro inesperado ao gerar modelo 3D.',
    success_generated: 'Molde 3D (.STL) gerado com sucesso!',
    download_filename: 'molde-moldaai.stl',
    sample_filename: 'exemplo-coracao.png',
  }
};

let currentLang = 'en';

function t(key, ...args) {
  const dict = translations[currentLang] || translations.en;
  const val = dict[key];
  if (typeof val === 'function') {
    return val(...args);
  }
  return val !== undefined ? val : key;
}

// Global state
let currentImageBitmap = null;
let currentStlBlob = null;
let currentMesh = null;
let currentFacetCount = 0;
let isGenerating = false;

// DOM Elements
const langBtnEn = document.getElementById('lang-btn-en');
const langBtnPt = document.getElementById('lang-btn-pt');

const imageInput = document.getElementById('image-input');
const dropzone = document.getElementById('dropzone');
const dropzonePrompt = document.getElementById('dropzone-prompt');
const previewWrapper = document.getElementById('preview-wrapper');
const previewCanvas = document.getElementById('preview-canvas');
const hiddenCanvas = document.getElementById('hidden-canvas');
const imageDimBadge = document.getElementById('image-dim-badge');

const thresholdSlider = document.getElementById('threshold-slider');
const thresholdVal = document.getElementById('threshold-val');
const invertCheck = document.getElementById('invert-check');
const btnUseSample = document.getElementById('btn-use-sample');

const inputSize = document.getElementById('input-size');
const inputBladeThick = document.getElementById('input-blade-thick');
const inputBladeHeight = document.getElementById('input-blade-height');
const inputBaseThick = document.getElementById('input-base-thick');
const inputBaseHeight = document.getElementById('input-base-height');
const smoothingSlider = document.getElementById('smoothing-slider');
const smoothingVal = document.getElementById('smoothing-val');

const presetSmall = document.getElementById('preset-small');
const presetMedium = document.getElementById('preset-medium');
const presetLarge = document.getElementById('preset-large');

const btnGenerate = document.getElementById('btn-generate');
const generateBtnText = document.getElementById('generate-btn-text');
const generateIcon = document.getElementById('generate-icon');
const generateSpinner = document.getElementById('generate-spinner');
const statusBanner = document.getElementById('status-banner');

const threeContainer = document.getElementById('three-container');
const emptyState = document.getElementById('empty-state');
const viewportLoading = document.getElementById('viewport-loading');
const btnDownload = document.getElementById('btn-download');
const modelStats = document.getElementById('model-stats');
const trianglesCount = document.getElementById('triangles-count');
const filesizeBadge = document.getElementById('filesize-badge');

// Language switching implementation
function applyLanguage(lang) {
  currentLang = lang;
  localStorage.setItem('moldaai_lang', lang);
  document.documentElement.lang = lang === 'pt' ? 'pt-BR' : 'en';

  // Toggle button active states
  if (lang === 'en') {
    langBtnEn.className = 'px-2.5 py-1 text-xs font-semibold rounded-lg bg-white shadow-sm text-sand-900 border border-sand-200 flex items-center space-x-1';
    langBtnPt.className = 'px-2.5 py-1 text-xs font-semibold rounded-lg text-sand-600 hover:text-sand-900 flex items-center space-x-1';
  } else {
    langBtnPt.className = 'px-2.5 py-1 text-xs font-semibold rounded-lg bg-white shadow-sm text-sand-900 border border-sand-200 flex items-center space-x-1';
    langBtnEn.className = 'px-2.5 py-1 text-xs font-semibold rounded-lg text-sand-600 hover:text-sand-900 flex items-center space-x-1';
  }

  // Update DOM elements with data-i18n
  document.querySelectorAll('[data-i18n]').forEach((el) => {
    const key = el.getAttribute('data-i18n');
    const val = t(key);
    if (val) {
      if (el.tagName === 'TITLE') {
        document.title = val;
      } else {
        el.textContent = val;
      }
    }
  });

  // Update dynamic elements
  if (isGenerating) {
    generateBtnText.textContent = t('generate_btn_processing');
  } else {
    generateBtnText.textContent = t('generate_btn');
  }

  if (currentFacetCount > 0) {
    const locale = lang === 'pt' ? 'pt-BR' : 'en-US';
    trianglesCount.textContent = `${currentFacetCount.toLocaleString(locale)} ${t('triangles_suffix')}`;
  }
}

langBtnEn.addEventListener('click', () => applyLanguage('en'));
langBtnPt.addEventListener('click', () => applyLanguage('pt'));

// Detect initial language: stored preference -> browser language -> English default
const savedLang = localStorage.getItem('moldaai_lang');
const browserLang = (navigator.language || '').toLowerCase().startsWith('pt') ? 'pt' : 'en';
applyLanguage(savedLang || browserLang || 'en');

// --- Three.js Scene Setup ---
const scene = new THREE.Scene();
scene.background = new THREE.Color(0x232433);

const camera = new THREE.PerspectiveCamera(
  45,
  threeContainer.clientWidth / threeContainer.clientHeight,
  0.1,
  1000
);
camera.position.set(0, -90, 80);

const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: false });
renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
renderer.setSize(threeContainer.clientWidth, threeContainer.clientHeight);
renderer.toneMapping = THREE.ACESFilmicToneMapping;
renderer.toneMappingExposure = 1.2;
threeContainer.appendChild(renderer.domElement);

const controls = new OrbitControls(camera, renderer.domElement);
controls.enableDamping = true;
controls.dampingFactor = 0.05;
controls.maxPolarAngle = Math.PI / 2 + 0.05; // Don't flip below bed
controls.target.set(0, 0, 6);
controls.update();

// Lights
const ambientLight = new THREE.AmbientLight(0xffffff, 0.9);
scene.add(ambientLight);

const dirLight1 = new THREE.DirectionalLight(0xfff5ea, 2.0);
dirLight1.position.set(60, -80, 100);
scene.add(dirLight1);

const dirLight2 = new THREE.DirectionalLight(0xddeeff, 1.0);
dirLight2.position.set(-60, 80, 50);
scene.add(dirLight2);

// 3D Printer Bed Grid (160x160mm)
const bedGrid = new THREE.GridHelper(160, 32, 0xc85a32, 0x474966);
bedGrid.rotation.x = Math.PI / 2; // Lie flat in XY plane (Z is up)
scene.add(bedGrid);

// Animation Loop
function animate() {
  requestAnimationFrame(animate);
  controls.update();
  renderer.render(scene, camera);
}
animate();

// Resize Observer
const resizeObserver = new ResizeObserver(() => {
  const width = threeContainer.clientWidth;
  const height = threeContainer.clientHeight;
  if (width > 0 && height > 0) {
    camera.aspect = width / height;
    camera.updateProjectionMatrix();
    renderer.setSize(width, height);
  }
});
resizeObserver.observe(threeContainer);

// --- Presets: Pequeno, Médio, Grande ---
presetSmall.addEventListener('click', () => {
  inputSize.value = '45.0';
  inputBladeThick.value = '0.6';
  inputBladeHeight.value = '8.0';
  inputBaseThick.value = '2.8';
  inputBaseHeight.value = '2.5';
  smoothingSlider.value = '0.35';
  smoothingVal.textContent = '0.35 mm';
  flashHighlight(presetSmall);
});

presetMedium.addEventListener('click', () => {
  inputSize.value = '70.0';
  inputBladeThick.value = '0.8';
  inputBladeHeight.value = '12.0';
  inputBaseThick.value = '3.2';
  inputBaseHeight.value = '3.0';
  smoothingSlider.value = '0.50';
  smoothingVal.textContent = '0.50 mm';
  flashHighlight(presetMedium);
});

presetLarge.addEventListener('click', () => {
  inputSize.value = '100.0';
  inputBladeThick.value = '0.8';
  inputBladeHeight.value = '15.0';
  inputBaseThick.value = '3.8';
  inputBaseHeight.value = '3.5';
  smoothingSlider.value = '0.60';
  smoothingVal.textContent = '0.60 mm';
  flashHighlight(presetLarge);
});

smoothingSlider.addEventListener('input', (e) => {
  smoothingVal.textContent = `${parseFloat(e.target.value).toFixed(2)} mm`;
});

function flashHighlight(btn) {
  btn.classList.add('ring-2', 'ring-clay-500');
  setTimeout(() => btn.classList.remove('ring-2', 'ring-clay-500'), 500);
}

// --- Image Loading & Client-side Pre-processing ---
imageInput.addEventListener('change', (e) => {
  const file = e.target.files[0];
  if (file) handleImageFile(file);
});

// Drag and drop handlers
dropzone.addEventListener('dragover', (e) => {
  e.preventDefault();
  dropzone.classList.add('border-clay-500', 'bg-clay-50/50');
});

dropzone.addEventListener('dragleave', () => {
  dropzone.classList.remove('border-clay-500', 'bg-clay-50/50');
});

dropzone.addEventListener('drop', (e) => {
  e.preventDefault();
  dropzone.classList.remove('border-clay-500', 'bg-clay-50/50');
  const file = e.dataTransfer.files[0];
  const isSupported = file && (
    file.type === 'image/png' ||
    file.type === 'image/jpeg' ||
    file.type === 'image/webp' ||
    file.type === 'image/svg+xml' ||
    file.name.toLowerCase().endsWith('.svg') ||
    file.name.toLowerCase().endsWith('.webp')
  );
  if (isSupported) {
    handleImageFile(file);
  } else {
    showError(t('err_invalid_image'));
  }
});

function handleImageFile(file) {
  const reader = new FileReader();
  reader.onload = (e) => {
    const img = new Image();
    img.onload = () => {
      processLoadedImage(img, file.name);
    };
    img.src = e.target.result;
  };
  reader.readAsDataURL(file);
}

// Draw a sample heart shape for instant testing
btnUseSample.addEventListener('click', () => {
  const size = 500;
  const canvas = document.createElement('canvas');
  canvas.width = size;
  canvas.height = size;
  const ctx = canvas.getContext('2d');

  // Fill with pure white
  ctx.fillStyle = '#FFFFFF';
  ctx.fillRect(0, 0, size, size);

  // Draw heart in black
  ctx.fillStyle = '#000000';
  ctx.beginPath();
  const topY = 160;
  ctx.moveTo(250, topY + 60);
  ctx.bezierCurveTo(250, topY + 20, 200, topY - 30, 140, topY - 30);
  ctx.bezierCurveTo(60, topY - 30, 60, topY + 80, 60, topY + 80);
  ctx.bezierCurveTo(60, topY + 160, 150, topY + 220, 250, topY + 290);
  ctx.bezierCurveTo(350, topY + 220, 440, topY + 160, 440, topY + 80);
  ctx.bezierCurveTo(440, topY + 80, 440, topY - 30, 360, topY - 30);
  ctx.bezierCurveTo(300, topY - 30, 250, topY + 20, 250, topY + 60);
  ctx.fill();

  const img = new Image();
  img.onload = () => {
    processLoadedImage(img, t('sample_filename'));
  };
  img.src = canvas.toDataURL('image/png');
});

function processLoadedImage(img, filename) {
  // Scale down to max 800px on largest dimension
  const maxDim = 800;
  let w = img.width;
  let h = img.height;

  if (w > maxDim || h > maxDim) {
    if (w > h) {
      h = Math.round((h * maxDim) / w);
      w = maxDim;
    } else {
      w = Math.round((w * maxDim) / h);
      h = maxDim;
    }
  }

  hiddenCanvas.width = w;
  hiddenCanvas.height = h;
  const ctx = hiddenCanvas.getContext('2d');
  ctx.clearRect(0, 0, w, h);
  ctx.drawImage(img, 0, 0, w, h);

  currentImageBitmap = ctx.getImageData(0, 0, w, h);

  // Update UI
  dropzonePrompt.classList.add('hidden');
  previewWrapper.classList.remove('hidden');
  imageDimBadge.classList.remove('hidden');
  imageDimBadge.textContent = `${w}×${h}px (${filename})`;

  hideStatus();
  update2DPreview();
}

// Live 2D Threshold Binarization Preview
thresholdSlider.addEventListener('input', (e) => {
  thresholdVal.textContent = e.target.value;
  update2DPreview();
});

invertCheck.addEventListener('change', () => {
  update2DPreview();
});

function update2DPreview() {
  if (!currentImageBitmap) return;

  const w = currentImageBitmap.width;
  const h = currentImageBitmap.height;
  const threshold = parseInt(thresholdSlider.value, 10);
  const invert = invertCheck.checked;

  previewCanvas.width = w;
  previewCanvas.height = h;
  const ctx = previewCanvas.getContext('2d');
  const imgData = ctx.createImageData(w, h);

  const src = currentImageBitmap.data;
  const dst = imgData.data;

  for (let i = 0; i < src.length; i += 4) {
    const r = src[i];
    const g = src[i + 1];
    const b = src[i + 2];
    const a = src[i + 3];

    // Transparent pixel treated as background
    let isSolid = false;
    if (a >= 128) {
      const lum = 0.299 * r + 0.587 * g + 0.114 * b;
      isSolid = lum < threshold;
    }

    if (invert) {
      isSolid = !isSolid;
    }

    const val = isSolid ? 0 : 255;
    dst[i] = val;
    dst[i + 1] = val;
    dst[i + 2] = val;
    dst[i + 3] = 255;
  }

  ctx.putImageData(imgData, 0, 0);
}

// --- Generate STL Action ---
btnGenerate.addEventListener('click', async () => {
  if (!currentImageBitmap) {
    showError(t('err_no_image'));
    return;
  }

  const baseHeight = parseFloat(inputBaseHeight.value);
  const bladeHeight = parseFloat(inputBladeHeight.value);
  const baseThick = parseFloat(inputBaseThick.value);
  const bladeThick = parseFloat(inputBladeThick.value);

  if (baseHeight >= bladeHeight) {
    showError(t('err_height_order', baseHeight, bladeHeight));
    return;
  }
  if (baseThick <= bladeThick) {
    showError(t('err_thick_order', baseThick, bladeThick));
    return;
  }

  setLoading(true);
  hideStatus();

  try {
    // Convert preview canvas directly to PNG Blob
    const blob = await new Promise((resolve) => previewCanvas.toBlob(resolve, 'image/png'));
    if (!blob) throw new Error(t('err_export_blob'));

    const formData = new FormData();
    formData.append('image', blob, 'silhueta.png');
    formData.append('size_mm', inputSize.value);
    formData.append('blade_thickness_mm', inputBladeThick.value);
    formData.append('blade_height_mm', inputBladeHeight.value);
    formData.append('base_thickness_mm', inputBaseThick.value);
    formData.append('base_height_mm', inputBaseHeight.value);
    formData.append('smoothing_mm', smoothingSlider.value);
    formData.append('threshold', '128'); // Already binarized at 128
    formData.append('invert', 'false');

    const response = await fetch('/api/generate', {
      method: 'POST',
      body: formData,
    });

    if (!response.ok) {
      let errorMsg = `HTTP ${response.status}`;
      try {
        const errJson = await response.json();
        if (errJson && errJson.error) errorMsg = errJson.error;
      } catch (_) {}
      throw new Error(errorMsg);
    }

    const arrayBuffer = await response.arrayBuffer();
    currentStlBlob = new Blob([arrayBuffer], { type: 'application/sla' });

    // Render 3D Model in Three.js
    loadStlIntoScene(arrayBuffer);

    // Enable download button
    btnDownload.disabled = false;
    showSuccess(t('success_generated'));
  } catch (err) {
    showError(err.message || t('err_unexpected'));
  } finally {
    setLoading(false);
  }
});

// Render STL in Three.js Viewport
function loadStlIntoScene(arrayBuffer) {
  const loader = new STLLoader();
  const geometry = loader.parse(arrayBuffer);

  // Compute bounding box and center model
  geometry.computeBoundingBox();
  geometry.center();
  geometry.computeVertexNormals();

  // Remove previous mesh
  if (currentMesh) {
    scene.remove(currentMesh);
    if (currentMesh.geometry) currentMesh.geometry.dispose();
    if (currentMesh.material) currentMesh.material.dispose();
  }

  // Material with terracotta PLA plastic aesthetic
  const material = new THREE.MeshStandardMaterial({
    color: 0xc85a32,
    roughness: 0.38,
    metalness: 0.08,
  });

  currentMesh = new THREE.Mesh(geometry, material);

  // Lift mesh so bottom sits exactly on top of print bed (Z=0)
  const bbox = geometry.boundingBox;
  const zHeight = bbox.max.z - bbox.min.z;
  currentMesh.position.z = zHeight / 2;

  scene.add(currentMesh);

  // Adjust camera to fit cutter comfortably
  const maxDim = Math.max(bbox.max.x - bbox.min.x, bbox.max.y - bbox.min.y);
  camera.position.set(0, -maxDim * 1.6, maxDim * 1.2);
  controls.target.set(0, 0, zHeight / 2);
  controls.update();

  // Update Stats in UI
  emptyState.classList.add('hidden');
  modelStats.classList.remove('hidden');
  currentFacetCount = geometry.attributes.position.count / 3;
  const locale = currentLang === 'pt' ? 'pt-BR' : 'en-US';
  trianglesCount.textContent = `${currentFacetCount.toLocaleString(locale)} ${t('triangles_suffix')}`;
  filesizeBadge.textContent = `${(arrayBuffer.byteLength / 1024).toFixed(1)} KB`;
}

// Download STL
btnDownload.addEventListener('click', () => {
  if (!currentStlBlob) return;
  const url = URL.createObjectURL(currentStlBlob);
  const a = document.createElement('a');
  a.href = url;
  a.download = t('download_filename');
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  setTimeout(() => URL.revokeObjectURL(url), 1000);
});

// UI Helpers
function setLoading(loading) {
  isGenerating = loading;
  btnGenerate.disabled = loading;
  if (loading) {
    generateIcon.classList.add('hidden');
    generateSpinner.classList.remove('hidden');
    generateBtnText.textContent = t('generate_btn_processing');
    viewportLoading.classList.remove('hidden');
  } else {
    generateIcon.classList.remove('hidden');
    generateSpinner.classList.add('hidden');
    generateBtnText.textContent = t('generate_btn');
    viewportLoading.classList.add('hidden');
  }
}

function showError(msg) {
  statusBanner.className = 'p-3.5 rounded-xl text-xs font-medium flex items-start space-x-2 bg-red-50 text-red-800 border border-red-200';
  statusBanner.innerHTML = `
    <svg class="w-4 h-4 text-red-600 mt-0.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
    <span>${msg}</span>
  `;
  statusBanner.classList.remove('hidden');
}

function showSuccess(msg) {
  statusBanner.className = 'p-3.5 rounded-xl text-xs font-medium flex items-start space-x-2 bg-emerald-50 text-emerald-800 border border-emerald-200';
  statusBanner.innerHTML = `
    <svg class="w-4 h-4 text-emerald-600 mt-0.5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
    </svg>
    <span>${msg}</span>
  `;
  statusBanner.classList.remove('hidden');
}

function hideStatus() {
  statusBanner.classList.add('hidden');
}
