type Domain = {
	domain_name: string;
	domain_name_formatted: string;
	domain_sort_order: number;
};

type DomainCache = {
	timestamp: number;
	domains: Domain[];
};

type ManufacturersAndModels = {
	system_model: string;
	system_manufacturer: string;
};

type ManufacturerAndModelsCache = {
	timestamp: number;
	manufacturers_and_models: ManufacturersAndModels[];
};

type FilterParams = {
	inputElement: HTMLSelectElement;
	resetElement: HTMLElement;
	paramString: string;
};

const inventoryFilterForm = document.getElementById('inventory-search-form') as HTMLFormElement;
const inventoryFilterFormResetButton = document.getElementById('inventory-search-form-reset-button') as HTMLElement;
const filterLocation = document.getElementById('inventory-search-location') as HTMLSelectElement;
const filterLocationReset = document.getElementById('inventory-search-location-reset') as HTMLElement;
const filterDepartment = document.getElementById('inventory-search-department') as HTMLSelectElement;
const filterDepartmentReset = document.getElementById('inventory-search-department-reset') as HTMLElement;
const filterManufacturer = document.getElementById('inventory-search-manufacturer') as HTMLSelectElement;
const filterManufacturerReset = document.getElementById('inventory-search-manufacturer-reset') as HTMLElement;
const filterModel = document.getElementById('inventory-search-model') as HTMLSelectElement;
const filterModelReset = document.getElementById('inventory-search-model-reset') as HTMLElement;
const filterDomain = document.getElementById('inventory-search-domain') as HTMLSelectElement;
const filterDomainReset = document.getElementById('inventory-search-domain-reset') as HTMLElement;
const filterStatus = document.getElementById('inventory-search-status') as HTMLSelectElement;
const filterStatusReset = document.getElementById('inventory-search-status-reset') as HTMLElement;
const filterBroken = document.getElementById('inventory-search-broken') as HTMLSelectElement;
const filterBrokenReset = document.getElementById('inventory-search-broken-reset') as HTMLElement;
const filterHasImages = document.getElementById('inventory-search-has_images') as HTMLSelectElement;
const filterHasImagesReset = document.getElementById('inventory-search-has_images-reset') as HTMLElement;

const urlSearchParams: FilterParams[] = [
	{ inputElement: filterLocation, resetElement: filterLocationReset, paramString: 'location' },
	{ inputElement: filterDepartment, resetElement: filterDepartmentReset, paramString: 'department_name' },
	{ inputElement: filterManufacturer, resetElement: filterManufacturerReset, paramString: 'system_manufacturer' },
	{ inputElement: filterModel, resetElement: filterModelReset, paramString: 'system_model' },
	{ inputElement: filterDomain, resetElement: filterDomainReset, paramString: 'ad_domain' },
	{ inputElement: filterStatus, resetElement: filterStatusReset, paramString: 'status' },
	{ inputElement: filterBroken, resetElement: filterBrokenReset, paramString: 'is_broken' },
	{ inputElement: filterHasImages, resetElement: filterHasImagesReset, paramString: 'has_images' }
];

let allModelsData: string[] = [];

function initializeSearch() {
	updateURLFromFilters();

	createFilterResetHandler(filterLocation, filterLocationReset);
	createFilterResetHandler(filterDepartment, filterDepartmentReset);
	createFilterResetHandler(filterManufacturer, filterManufacturerReset);
	createFilterResetHandler(filterModel, filterModelReset);
	createFilterResetHandler(filterDomain, filterDomainReset);
	createFilterResetHandler(filterStatus, filterStatusReset);
	createFilterResetHandler(filterBroken, filterBrokenReset);
	createFilterResetHandler(filterHasImages, filterHasImagesReset);

	filterModel.disabled = !filterManufacturer.value;
}

function resetSearchURLParameters() {
	for (const param of urlSearchParams) {
		if (!param.paramString) continue;
		setURLParameter(param.paramString, null);
	}
}

function updateFiltersFromURL() {
	const currentParams = new URLSearchParams(window.location.search);
	for (const param of urlSearchParams) {
		if (!param.inputElement || !param.paramString) continue;
		const urlValue = currentParams.get(param.paramString);
		if (urlValue && urlValue.trim().length > 0) {
			param.inputElement.value = urlValue;
			param.inputElement.style.border = "1px solid orange";
			param.inputElement.style.outline = "1px solid orange;"
			param.inputElement.style.boxShadow = "0 0 2px orange";
			param.resetElement.style.display = 'inline-block';
		} else {
			param.inputElement.value = '';
			param.inputElement.style.border = "revert-layer";
			param.inputElement.style.boxShadow = "revert-layer";
			param.inputElement.style.outline = "revert-layer";
			param.resetElement.style.display = 'none';
		}
	}
}

function createFilterResetHandler(filterElement: HTMLSelectElement, resetButton: HTMLElement) {
	if (!filterElement || !resetButton) {
		console.error("Filter inputElement or reset button not found.");
		return;
	}

	if (filterElement.value && filterElement.value.length > 0) {
		resetButton.style.display = 'inline-block';
		filterElement.style.border = "1px solid orange";
		filterElement.style.outline = "1px solid orange;"
		filterElement.style.boxShadow = "0 0 2px orange";
	}

	filterElement.addEventListener("change", () => {
		resetButton.style.display = 'inline-block';
		const paramString = getURLParamName(filterElement);
		setURLParameter(paramString, filterElement.value);
		if ((filterElement.value && filterElement.value.trim().length >= 0) || typeof filterElement.value === 'boolean') {
			resetButton.style.display = 'inline-block';
			filterElement.style.border = "1px solid orange";
			filterElement.style.outline = "1px solid orange;"
			filterElement.style.boxShadow = "0 0 2px orange";
		} else {
			resetButton.style.display = 'none';
			filterElement.style.border = "revert-layer";
			filterElement.style.boxShadow = "revert-layer";
		}
		if (filterElement === filterManufacturerReset || filterElement == filterModelReset) {
			populateManufacturerSelect().catch((error) => {
				console.error("Error populating manufacturer select:", error);
			});
			populateModelSelect().catch((error) => {
				console.error("Error populating model select:", error);
			});
		}
		fetchFilteredInventoryData().catch((error) => {
			console.error("Error fetching filtered inventory data:", error);
		});
	});
  
	resetButton.addEventListener("click", (event) => {
		event.preventDefault();
		resetButton.style.display = 'none';
		filterElement.style.border = "revert-layer";
		filterElement.style.boxShadow = "revert-layer";
		filterElement.style.outline = "revert-layer";
		filterElement.value = '';
		if (resetButton === filterManufacturerReset || resetButton == filterModelReset) {
			populateManufacturerSelect().catch((error) => {
				console.error("Error populating manufacturer select:", error);
			});
			populateModelSelect().catch((error) => {
				console.error("Error populating model select:", error);
			});
		}
		for (const elem of urlSearchParams) {
			if (elem.inputElement === filterElement) {
				setURLParameter(elem.paramString, null);
				break;
			}
		}
		fetchFilteredInventoryData().catch((error) => {
			console.error("Error fetching filtered inventory data:", error);
		});
	});
}

async function fetchFilteredInventoryData(csvDownload = false): Promise<void> {
	const currentParams = new URLSearchParams(window.location.search);

	setURLParameter('update', currentParams.get('update')?.trim() || null);
	setURLParameter('tagnumber', currentParams.get('tagnumber')?.trim() || null);
	setURLParameter('system_serial', currentParams.get('system_serial')?.trim() || null);

	updateURLFromFilters();

	const apiQuery = new URLSearchParams(currentParams); // API query parameters
	if (currentParams.get('update') === "true") {
		apiQuery.delete("update");
		apiQuery.delete("tagnumber");
		apiQuery.delete("system_serial");
	}

	if (csvDownload) {
		window.location.href = `/api/inventory?csv=true&${apiQuery.toString()}`;
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

async function fetchAllManufacturersAndModels(purgeCache: boolean = false): Promise<Array<ManufacturersAndModels> | []> {
	const cached = sessionStorage.getItem("uit_manufacturers_and_models");

  try {
		if (cached && !purgeCache) {
			const cacheEntry: ManufacturerAndModelsCache = JSON.parse(cached);
			if (Date.now() - cacheEntry.timestamp < 300000 && Array.isArray(cacheEntry.manufacturers_and_models)) {
				console.log("Loaded manufacturers and models from cache");
				return cacheEntry.manufacturers_and_models;
			}
		}

    const data: ManufacturersAndModels[] = await fetchData('/api/models');
    if (!data || !Array.isArray(data) || data.length === 0) {
      throw new Error('No data returned from /api/models');
    }
		const cacheEntry: ManufacturerAndModelsCache = {
			timestamp: Date.now(),
			manufacturers_and_models: data
		};
		sessionStorage.setItem("uit_manufacturers_and_models", JSON.stringify(cacheEntry));
		console.log("Cached manufacturers and models data");
    return data;
  } catch (error) {
    console.error('Error fetching manufacturers and models:', error);
		return [];
  }
}

async function populateManufacturerSelect(purgeCache: boolean = false) {
  if (!filterManufacturer) return;

	const initialValue = filterManufacturer.value;

	filterManufacturer.disabled = true;

  const data: ManufacturersAndModels[] = await fetchAllManufacturersAndModels(purgeCache);
	if (!data || !Array.isArray(data) || data.length === 0) return;

	// Sort manufacturers array - get unique key
	const uniqueMap = new Map<string, ManufacturersAndModels>();
	for (const item of data) {
		if (!item.system_manufacturer) continue;
		if (!uniqueMap.has(item.system_manufacturer)) {
			uniqueMap.set(item.system_manufacturer, item);
		}
	}
	const uniqueArray = Array.from(uniqueMap.values());
	uniqueArray.sort((a, b) => {
		const manufacturerA = a.system_manufacturer || a.system_manufacturer;
		const manufacturerB = b.system_manufacturer || b.system_manufacturer;
		return manufacturerA.localeCompare(manufacturerB);
	});

  // Clear and rebuild manufacturer select options
  resetSelectElement(filterManufacturer, 'Manufacturer');

  // Sort by formatted name
  for (const item of uniqueArray) {
		if (!item.system_manufacturer || !item.system_manufacturer) continue;
		const option = document.createElement('option');
		option.value = item.system_manufacturer;
		option.textContent = item.system_manufacturer || item.system_manufacturer;
		filterManufacturer.appendChild(option);
	}

  filterManufacturer.value = (initialValue && uniqueArray.some(item => item.system_manufacturer === initialValue)) ? initialValue : '';
	if (filterManufacturer.value !== '') {
		setURLParameter('system_manufacturer', filterManufacturer.value);
		filterModel.disabled = false;
	} else {
		setURLParameter('system_manufacturer', null);
	}
	filterManufacturer.disabled = false;
}

async function populateModelSelect(purgeCache: boolean = false) {
  if (!filterModel) return;
	
	const initialValue = filterModel.value;
	
	filterModel.disabled = true;

	if (!filterManufacturer.value || filterManufacturer.value.trim().length === 0) {
		resetSelectElement(filterModel, 'Model', true);
		setURLParameter('system_model', null);
		return;
	}

  const data: ManufacturersAndModels[] = await fetchAllManufacturersAndModels(purgeCache);
	if (!data || !Array.isArray(data) || data.length === 0) return;

	data.sort((a, b) => {
		const modelA = a.system_model || a.system_model;
		const modelB = b.system_model || b.system_model;
		return modelA.localeCompare(modelB);
	});

	resetSelectElement(filterModel, 'Model');

	for (const item of data) {
		if (item.system_manufacturer !== filterManufacturer.value) {
			console.log("Skipping model for manufacturer:", item.system_model, item.system_manufacturer, filterManufacturer.value);
			continue;
		};
		if (!item.system_model || !item.system_model) continue;
		const option = document.createElement('option');
		option.value = item.system_model;
		option.textContent = item.system_model || item.system_model;
		filterModel.appendChild(option);
	}

	filterModel.value = (initialValue && data.some(item => item.system_model === initialValue)) ? initialValue : '';
	filterModel.disabled = false;
}

async function fetchDomains(purgeCache: boolean = false): Promise<Array<Domain> | []> {
	const cached = sessionStorage.getItem("uit_domains");

	try {
		if (cached && !purgeCache) {
			const cacheEntry: DomainCache = JSON.parse(cached);
			if (Date.now() - cacheEntry.timestamp < 300000 && Array.isArray(cacheEntry.domains)) {
				console.log("Loaded domains from cache");
				return cacheEntry.domains;
			}
		}
		const data: Array<Domain> = await fetchData('/api/domains');
		if (!data || !Array.isArray(data) || data.length === 0) {
			throw new Error('No data returned from /api/domains');
		}
		const cacheEntry: DomainCache = {
			timestamp: Date.now(),
			domains: data
		};
		sessionStorage.setItem("uit_domains", JSON.stringify(cacheEntry));
		console.log("Cached domains data");
		return data;
	} catch (error) {
		console.error('Error fetching domains:', error);
		return [];
	}
}

async function populateDomainSelect(elem: HTMLSelectElement, purgeCache: boolean = false) {
	if (!elem) return;

	const initialValue = elem.value;

	elem.disabled = true;

	try {
		const domainData: Array<Domain> = await fetchDomains(purgeCache);
		if (!domainData || !Array.isArray(domainData) || domainData.length === 0) {
			throw new Error('No data returned from /api/domains');
		}

		domainData.sort((a, b) => {
			return a.domain_sort_order - b.domain_sort_order;
		});

		resetSelectElement(elem, 'Domain');

		for (const domain of domainData) {
			const option = document.createElement('option');
			option.value = domain.domain_name;
			option.textContent = domain.domain_name_formatted || domain.domain_name;
			elem.appendChild(option);
		}

		elem.value = (initialValue && domainData.some(item => item.domain_name === initialValue)) ? initialValue : '';
	} catch (error) {
		console.error('Error fetching domains:', error);
	} finally {
		elem.disabled = false;
	}
}

async function populateDepartmentSelect(elem: HTMLSelectElement, purgeCache: boolean = false) {
	if (!elem) return;

	const initialValue = elem.value;

	elem.disabled = true;

	try {
		const departmentsData: Array<Department> = await fetchDepartments(purgeCache);
		if (!departmentsData || !Array.isArray(departmentsData) || departmentsData.length === 0) {
			throw new Error('No data returned from /api/departments');
		}

		departmentsData.sort((a, b) => {
			return a.department_sort_order - b.department_sort_order;
		});

		resetSelectElement(elem, 'Department');


		for (const department of departmentsData) {
			const option = document.createElement('option');
			option.value = department.department_name;
			option.textContent = department.department_name_formatted || department.department_name;
			elem.appendChild(option);
		}
		elem.value = (initialValue && departmentsData.some(item => item.department_name === initialValue)) ? initialValue : '';
	} catch (error) {
		console.error('Error fetching departments:', error);
	} finally {
		elem.disabled = false;
	}
}

inventoryFilterForm.addEventListener("submit", (event) => {
  event.preventDefault();
  fetchFilteredInventoryData();
});

inventoryFilterFormResetButton.addEventListener("click", async (event) => {
  event.preventDefault();
  document.querySelectorAll('#inventory-search-form input').forEach((input: HTMLInputElement) => {
    input.value = '';
		input.disabled = true;
  });

	for (const param of urlSearchParams) {
		if (!param.inputElement || !param.paramString) continue;
		param.inputElement.style.border = "revert-layer";
		param.inputElement.style.boxShadow = "revert-layer";
		param.inputElement.style.outline = "revert-layer";
		param.resetElement.style.display = 'none';
		param.inputElement.value = '';
	}

	resetSearchURLParameters();
	try{
		await populateDepartmentSelect(filterDepartment);
		await populateManufacturerSelect();
		await populateModelSelect();
		await populateDomainSelect(filterDomain);
		await fetchFilteredInventoryData();
	} catch (error) {
		console.error("Error resetting filters and fetching data:", error);
	}
});