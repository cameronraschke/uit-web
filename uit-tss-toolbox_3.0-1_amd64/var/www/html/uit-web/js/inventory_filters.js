const filterLocation = document.getElementById('inventory-filter-location');
const filterLocationReset = document.getElementById('inventory-filter-location-reset');
const filterDepartment = document.getElementById('inventory-filter-department');
const filterDepartmentReset = document.getElementById('inventory-filter-department-reset');
const filterManufacturer = document.getElementById('inventory-filter-manufacturer');
const filterManufacturerReset = document.getElementById('inventory-filter-manufacturer-reset');
const filterModel = document.getElementById('inventory-filter-model');
const filterModelReset = document.getElementById('inventory-filter-model-reset');
const filterDomain = document.getElementById('inventory-filter-domain');
const filterDomainReset = document.getElementById('inventory-filter-domain-reset');
const filterStatus = document.getElementById('inventory-filter-status');
const filterStatusReset = document.getElementById('inventory-filter-status-reset');
const filterBroken = document.getElementById('inventory-filter-broken');
const filterBrokenReset = document.getElementById('inventory-filter-broken-reset');
const filterHasImages = document.getElementById('inventory-filter-has_images');
const filterHasImagesReset = document.getElementById('inventory-filter-has_images-reset');

let allModelsData = [];

const currentURL = new URL(window.location.href);
const queryParams = new URLSearchParams(currentURL.search);

function updateURLParameters(urlParameter, value) {
	const newURL = new URL(window.location.href);
	if (value) {
		newURL.searchParams.set(urlParameter, value);
	} else {
		newURL.searchParams.delete(urlParameter);
	}
	if (newURL.searchParams.toString()) {
		history.replaceState(null, '', newURL.pathname + '?' + newURL.searchParams.toString());
	} else {
		history.replaceState(null, '', newURL.pathname);
	}
}

async function initializeSearch() {
	await setFiltersFromURL();

	createFilterResetHandler(filterLocation, filterLocationReset);
	createFilterResetHandler(filterDepartment, filterDepartmentReset);
	createFilterResetHandler(filterManufacturer, filterManufacturerReset);
	createFilterResetHandler(filterModel, filterModelReset);
	createFilterResetHandler(filterDomain, filterDomainReset);
	createFilterResetHandler(filterStatus, filterStatusReset);
	createFilterResetHandler(filterBroken, filterBrokenReset);
	createFilterResetHandler(filterHasImages, filterHasImagesReset);

	if (filterLocation.value) filterLocationReset.style.display = 'inline-block';
	if (filterDepartment.value) filterDepartmentReset.style.display = 'inline-block';
	if (filterManufacturer.value) filterManufacturerReset.style.display = 'inline-block';
	if (filterModel.value) filterModelReset.style.display = 'inline-block';
	if (filterDomain.value) filterDomainReset.style.display = 'inline-block';
	if (filterStatus.value) filterStatusReset.style.display = 'inline-block';
	if (filterBroken.value) filterBrokenReset.style.display = 'inline-block';
	if (filterHasImages.value) filterHasImagesReset.style.display = 'inline-block';
}

// Reset filter
function createFilterResetHandler(filterElement, resetButton) {
	if (!filterElement || !resetButton) return;
	if (filterElement.value && filterElement.value.length > 0) {
		resetButton.style.display = 'inline-block';
	}

	filterElement.addEventListener("change", async () => {
		resetButton.style.display = 'inline-block';
		const paramName = getURLParamName(filterElement);
		updateURLParameters(paramName, filterElement.value);
		if (filterElement === filterManufacturer) {
			await populateModelSelect(filterElement.value || null);
		}
		await fetchFilteredInventoryData();
	});
  
	resetButton.addEventListener("click", async (event) => {
		event.preventDefault();
		resetButton.style.display = 'none';
		filterElement.value = '';
		const paramName = getURLParamName(filterElement);
		updateURLParameters(paramName, null);
		if (filterElement === filterManufacturer) {
			filterModel.value = '';
			filterModelReset.style.display = 'none';
			updateURLParameters('system_model', null);
			await loadAllManufacturersAndModels();
		}
		if (filterElement === filterModel) {
			await populateModelSelect(filterManufacturer.value || null);
		}
		await fetchFilteredInventoryData();
	});
}

function getURLParamName(filterElement) {
	if (filterElement === filterLocation) return 'location';
	if (filterElement === filterDepartment) return 'department_name';
	if (filterElement === filterManufacturer) return 'system_manufacturer';
	if (filterElement === filterModel) return 'system_model';
	if (filterElement === filterDomain) return 'ad_domain';
	if (filterElement === filterStatus) return 'status';
	if (filterElement === filterBroken) return 'is_broken';
	if (filterElement === filterHasImages) return 'has_images';
	return '';
}

async function setFiltersFromURL() {
	const currentParams = new URLSearchParams(window.location.search);
	filterLocation.value = currentParams.get('location') || '';
	filterDepartment.value = currentParams.get('department_name') || '';
	filterManufacturer.value = currentParams.get('system_manufacturer') || '';
	filterModel.value = currentParams.get('system_model') || '';
	filterDomain.value = currentParams.get('ad_domain') || '';
	filterStatus.value = currentParams.get('status') || '';
	filterBroken.value = currentParams.get('is_broken') || '';
	filterHasImages.value = currentParams.get('has_images') || '';
}

async function fetchFilteredInventoryData(csvDownload = false) {
	const currentParams = new URLSearchParams(window.location.search);

	const update = currentParams.get('update');
	const tagnumber = currentParams.get('tagnumber') || null;
	const systemSerial = currentParams.get('system_serial') || null;

	const location = filterLocation.value.trim() || null;
	const department = filterDepartment.value.trim() || null;
	const manufacturer = filterManufacturer.value.trim() || null;
	const model = filterModel.value.trim() || null;
	const domain = filterDomain.value.trim() || null;
	const status = filterStatus.value.trim() || null;
	const broken = filterBroken.value.trim() || null;
	const hasImages = filterHasImages.value.trim() || null;

	const browserQuery = new URLSearchParams();
	if (update) browserQuery.set('update', update);
	if (tagnumber) browserQuery.set('tagnumber', tagnumber);
	if (systemSerial) browserQuery.set('system_serial', systemSerial);
	if (location) browserQuery.set('location', location);
	if (department) browserQuery.set('department_name', department);
	if (manufacturer) browserQuery.set('system_manufacturer', manufacturer);
	if (model) browserQuery.set('system_model', model);
	if (domain) browserQuery.set('ad_domain', domain);
	if (status) browserQuery.set('status', status);
	if (broken) browserQuery.set('is_broken', broken);
	if (hasImages) browserQuery.set('has_images', hasImages);

	const apiQuery = new URLSearchParams(browserQuery); // Copy
	if (update === "true") {
		apiQuery.delete("update");
		apiQuery.delete("tagnumber");
		apiQuery.delete("system_serial");
	}

	const newURL = new URL(window.location.href);
	newURL.search = browserQuery.toString();
	if (browserQuery.toString()) {
		history.replaceState(null, '', newURL.pathname + '?' + browserQuery.toString());
	} else {
		history.replaceState(null, '', newURL.pathname);
	}

	if (csvDownload) {
		location.href = `/api/inventory?${apiQuery.toString()}`;
		return;
	}

  try {
    const response = await fetch(`/api/inventory?${apiQuery.toString()}`);
    const rawData = await response.text();
    const jsonData = rawData.trim() ? JSON.parse(rawData) : [];
    if (jsonData && typeof jsonData === 'object' && !Array.isArray(jsonData) && Object.prototype.hasOwnProperty.call(jsonData, 'error')) {
      throw new Error(String(jsonData.error || 'Unknown server error'));
    }
    await renderInventoryTable(jsonData);
  } catch (error) {
    console.error("Error fetching inventory data:", error);
  }
}

const inventoryFilterForm = document.getElementById('inventory-filter-form');
inventoryFilterForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  fetchFilteredInventoryData();
});

const inventoryFilterResetButton = document.getElementById('inventory-filter-form-reset-button');
inventoryFilterResetButton.addEventListener("click", async (event) => {
  event.preventDefault();
	history.replaceState(null, '', window.location.pathname);
  inventoryFilterForm.reset();
  document.querySelectorAll('.inventory-filter-reset').forEach(elem => {
    elem.style.display = 'none';
  });
	currentURL.search = '';
	await loadAllManufacturersAndModels();
  await fetchFilteredInventoryData();
});

async function populateManufacturerSelect() {
  const manufacturerSelect = document.getElementById('inventory-filter-manufacturer');
  if (!manufacturerSelect) return;

	const savedValue = manufacturerSelect.value;

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

	if (savedValue && manufacturersMap.has(savedValue)) {
    manufacturerSelect.value = savedValue;
  } else {
    manufacturerSelect.value = '';
  }
}

async function populateModelSelect(selectedManufacturer = null) {
  const modelSelect = document.getElementById('inventory-filter-model');
  if (!modelSelect) return;

	const savedValue = modelSelect.value;

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

	if (savedValue && modelsMap.has(savedValue)) {
    modelSelect.value = savedValue;
  } else {
    modelSelect.value = '';
  }
}

async function loadAllManufacturersAndModels() {
  try {
    const response = await fetchData('/api/models');
    if (!response) {
      throw new Error('No data returned from /api/models');
    }

    allModelsData = Array.isArray(response) ? response : [];
    await populateManufacturerSelect();
    await populateModelSelect();

  } catch (error) {
    console.error('Error fetching models:', error);
  }
}