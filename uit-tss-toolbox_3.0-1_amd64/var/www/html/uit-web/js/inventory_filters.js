let allModelsData = [];

// Filter form
const filterTag = document.getElementById('inventory-filter-tagnumber')
const filterSerial = document.getElementById('inventory-filter-serial')

// Location filter
const filterLocation = document.getElementById('inventory-filter-location')
const filterLocationReset = document.getElementById('inventory-filter-location-reset')
filterLocation.addEventListener("change", async (event) => {
  document.getElementById('inventory-filter-location-reset').style.display = 'inline-block';
  await fetchFilteredInventoryData();
});
filterLocationReset.addEventListener("click", async (event) => {
  event.preventDefault();
  filterLocationReset.style.display = 'none';
  filterLocation.value = '';
  await fetchFilteredInventoryData();
});
// Department filter
const filterDepartment = document.getElementById('inventory-filter-department')
const filterDepartmentReset = document.getElementById('inventory-filter-department-reset')
filterDepartment.addEventListener("change", async (event) => {
  filterDepartmentReset.style.display = 'inline-block';
  await fetchFilteredInventoryData();
});
filterDepartmentReset.addEventListener("click", async (event) => {
  event.preventDefault();
  filterDepartmentReset.style.display = 'none';
  filterDepartment.value = '';
  await fetchFilteredInventoryData();
});

// Manufacturer & model filters
const filterManufacturer = document.getElementById('inventory-filter-manufacturer')
const filterManufacturerReset = document.getElementById('inventory-filter-manufacturer-reset')
const filterModel = document.getElementById('inventory-filter-model')
const filterModelReset = document.getElementById('inventory-filter-model-reset')

filterManufacturer.addEventListener("change", async (event) => {
  filterManufacturerReset.style.display = 'inline-block';
  populateModelSelect(filterManufacturer.value || null);
  await updateModelOptionsBasedOnManufacturer();
  await fetchFilteredInventoryData();
});
filterManufacturerReset.addEventListener("click", async (event) => {
  event.preventDefault();
  filterManufacturerReset.style.display = 'none';
  filterManufacturer.value = '';
  // Also reset model select
  filterModelReset.style.display = 'none';
  filterModel.value = '';
  await loadManufacturersAndModels();
  await fetchFilteredInventoryData();
});

// Model filter
filterModel.addEventListener("change", async (event) => {
  filterModelReset.style.display = 'inline-block';
  await fetchFilteredInventoryData();
});
filterModelReset.addEventListener("click", async (event) => {
  event.preventDefault();
  filterModelReset.style.display = 'none';
  filterModel.value = '';
  populateModelSelect();
  await fetchFilteredInventoryData();
});

// Domain filter
const filterDomain = document.getElementById('inventory-filter-domain')
const filterDomainReset = document.getElementById('inventory-filter-domain-reset')
filterDomain.addEventListener("change", async (event) => {
  filterDomainReset.style.display = 'inline-block';
  await fetchFilteredInventoryData();
});
filterDomainReset.addEventListener("click", async (event) => {
  event.preventDefault();
  filterDomainReset.style.display = 'none';
  filterDomain.value = '';
  await fetchFilteredInventoryData();
});

// Status filter
const filterStatus = document.getElementById('inventory-filter-status')
const filterStatusReset = document.getElementById('inventory-filter-status-reset')
filterStatus.addEventListener("change", async (event) => {
  filterStatusReset.style.display = 'inline-block';
  await fetchFilteredInventoryData();
});
filterStatusReset.addEventListener("click", async (event) => {
  event.preventDefault();
  filterStatusReset.style.display = 'none';
  filterStatus.value = '';
  await fetchFilteredInventoryData();
});

// Broken filter
const filterBroken = document.getElementById('inventory-filter-broken')
const filterBrokenReset = document.getElementById('inventory-filter-broken-reset')
filterBroken.addEventListener("change", async (event) => {
  document.getElementById('inventory-filter-broken-reset').style.display = 'inline-block';
  await fetchFilteredInventoryData();
});
filterBrokenReset.addEventListener("click", async (event) => {
  event.preventDefault();
  filterBrokenReset.style.display = 'none';
  filterBroken.value = '';
  await fetchFilteredInventoryData();
});

// Has Images filter
const filterHasImages = document.getElementById('inventory-filter-has_images')
const filterHasImagesReset = document.getElementById('inventory-filter-has_images-reset')
filterHasImages.addEventListener("change", async (event) => {
  document.getElementById('inventory-filter-has_images-reset').style.display = 'inline-block';
  await fetchFilteredInventoryData();
});
filterHasImagesReset.addEventListener("click", async (event) => {
  event.preventDefault();
  filterHasImagesReset.style.display = 'none';
  filterHasImages.value = '';
  await fetchFilteredInventoryData();
});

async function fetchFilteredInventoryData(csvDownload = false) {
  const tag = filterTag.value.trim() || null;
  const serial = filterSerial.value.trim() || null;
  const location = filterLocation.value.trim() || null;
  const department = filterDepartment.value.trim() || null;
  const manufacturer = filterManufacturer.value.trim() || null;
  const model = filterModel.value.trim() || null;
  const domain = filterDomain.value.trim() || null;
  const status = filterStatus.value.trim() || null;
  const broken = filterBroken.value.trim() || null;
  const hasImages = filterHasImages.value.trim() || null;

  try {
    const tableData = await getInventoryTableData(csvDownload, tag, serial, location, department, manufacturer, model, domain, status, broken, hasImages);
    await renderInventoryTable(tableData);
  } catch (error) {
    console.error("Error fetching inventory data:", error);
  }
}

const inventoryFilterForm = document.getElementById('inventory-filter-form');
inventoryFilterForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  await fetchFilteredInventoryData();
});

const inventoryFilterResetButton = document.getElementById('inventory-filter-form-reset-button');
inventoryFilterResetButton.addEventListener("click", async (event) => {
  event.preventDefault();
  inventoryFilterForm.reset();
  document.querySelectorAll('.inventory-filter-reset').forEach(elem => {
    elem.style.display = 'none';
  });
  await loadManufacturersAndModels();
  await fetchFilteredInventoryData();
});

function populateManufacturerSelect() {
  const manufacturerSelect = document.getElementById('inventory-filter-manufacturer');
  if (!manufacturerSelect) return;

  // Get manufacturers
  const manufacturersMap = new Map();
  allModelsData.forEach(item => {
    if (item.system_manufacturer && !manufacturersMap.has(item.system_manufacturer)) {
      manufacturersMap.set(item.system_manufacturer, item.system_manufacturer_formatted || item.system_manufacturer);
    }
  });

  // Clear and rebuild manufacturer select options
  manufacturerSelect.innerHTML = '';
  const defaultOption = document.createElement('option');
  defaultOption.value = '';
  defaultOption.textContent = 'Manufacturer';
  defaultOption.selected = true;
  defaultOption.disabled = true;
  manufacturerSelect.appendChild(defaultOption);

  // Sort by formatted name
  const sortedManufacturers = Array.from(manufacturersMap.entries()).sort((a, b) => 
    a[1].localeCompare(b[1])
  );

  sortedManufacturers.forEach(([manufacturer, manufacturerFormatted]) => {
    const option = document.createElement('option');
    option.value = manufacturer;
    option.textContent = manufacturerFormatted;
    manufacturerSelect.appendChild(option);
  });
}

function populateModelSelect(selectedManufacturer = null) {
  const modelSelect = document.getElementById('inventory-filter-model');
  if (!modelSelect) return;

  // Filter models by manufacturer if one is selected
  const filteredModels = selectedManufacturer
    ? allModelsData.filter(item => item.system_manufacturer === selectedManufacturer)
    : allModelsData;

  // Get models
  const modelsMap = new Map();
  filteredModels.forEach(item => {
    if (item.system_model && !modelsMap.has(item.system_model)) {
      modelsMap.set(item.system_model, item.system_model_formatted || item.system_model);
    }
  });

  // Clear and rebuild model select options
  modelSelect.innerHTML = '';
  const defaultOption = document.createElement('option');
  defaultOption.value = '';
  defaultOption.textContent = 'Model';
  defaultOption.selected = true;
  defaultOption.disabled = true;
  modelSelect.appendChild(defaultOption);

  // Sort by formatted name
  const sortedModels = Array.from(modelsMap.entries()).sort((a, b) => 
    a[1].localeCompare(b[1])
  );

  sortedModels.forEach(([model, modelFormatted]) => {
    const option = document.createElement('option');
    option.value = model;
    option.textContent = modelFormatted;
    modelSelect.appendChild(option);
  });
}


async function loadManufacturersAndModels() {
  try {
    const response = await fetchData('/api/models');
    if (!response) {
      throw new Error('No data returned from /api/models');
    }

    allModelsData = Array.isArray(response) ? response : [];
    populateManufacturerSelect();
    populateModelSelect();

  } catch (error) {
    console.error('Error fetching models:', error);
  }
}

async function updateModelOptionsBasedOnManufacturer() {
  const manufacturerSelect = document.getElementById('inventory-filter-manufacturer');
  const selectedManufacturer = manufacturerSelect.value || null;
  
  // Repopulate model select with filtered options
  populateModelSelect(selectedManufacturer);
}