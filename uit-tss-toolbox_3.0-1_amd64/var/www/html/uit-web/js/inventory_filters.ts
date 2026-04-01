function getAdvSearchParamName(filterElement: HTMLSelectElement): string {
	for (const paramName in advSearchParams) {
		const param = advSearchParams[paramName];
		if (param.inputElement === filterElement) {
			return paramName;
		}
	}
	return '';
}

function resetAdvSearchURLParameters() {
	for (const paramName in advSearchParams) {
		if (!paramName) continue;
		setURLParameter(paramName, null, false);
	}
}

// function updateFiltersFromURL() {
// 	const currentParams = new URLSearchParams(window.location.search);
// 	for (const paramName in advSearchParams) {
// 		const param = advSearchParams[paramName];
// 		if (!param.inputElement || paramName === '') continue;
// 		if (currentParams.has(paramName)) {
// 			const urlValue: AdvSearchOptionString = JSON.parse(currentParams.get(paramName) || 'null');
// 			if (urlValue && urlValue.param_value && urlValue.param_value.trim().length > 0) param.inputElement.value = urlValue.param_value;
// 			handleAdvSearchInputChange([param]);
// 		}
// 	}
// }

function handleAdvSearchInputChange(filterEls: AdvSearchFilterElement[]) {
	for (const filterEl of filterEls) {
		if (!filterEl || !filterEl.inputElement || !filterEl.resetElement) {
			console.warn("Filter element is missing input or reset element: ", filterEl);
			return;
		}

		const paramName = getAdvSearchParamName(filterEl.inputElement);
		const rawValue = filterEl.inputElement.value.trim();
		const isBooleanFilter = paramName === 'filter_is_broken' || paramName === 'filter_has_images';

		// Testing a string here, otherwise "false" would not show the reset button
		if (rawValue !== '') {
			const urlValue = {
				param_value: isBooleanFilter ? rawValue === 'true' : rawValue,
				not: (filterEl.negationElement && filterEl.negationElement.checked === true) ? true : null,
			};
			setURLParameter(paramName, JSON.stringify(urlValue), true);
			filterEl.resetElement.style.display = 'inline-block';
			filterEl.inputElement.classList.add('changed-input');
		} else {
			setURLParameter(paramName, null);
			filterEl.resetElement.style.display = 'none';
			filterEl.inputElement.classList.remove('changed-input');
		}
	}
}

function initializeAdvSearchListeners(filterEls: AdvSearchFilterElement[]) {
	for (const filterEl of filterEls) {
		filterEl.inputElement.addEventListener("change", async () => {
			handleAdvSearchInputChange([filterEl]);
			try {
				if (filterEl.inputElement === filterManufacturer || filterEl.inputElement === filterModel) {
					await Promise.all([populateManufacturerSelect(advSearchParams['filter_system_manufacturer'].inputElement).then(() => populateModelSelect(advSearchParams['filter_system_model'].inputElement)), renderInventoryTable()]);
				} else {
					await renderInventoryTable();
				}
			} catch (err) {
				console.error(`Error fetching data from filterEl on change event listener:`, err);
			}
		});

		if (filterEl.negationElement) {
			filterEl.negationElement.addEventListener("change", async () => {
				handleAdvSearchInputChange([filterEl]);
				try {
					if (filterEl.inputElement === filterManufacturer || filterEl.inputElement === filterModel) {
						await Promise.all([populateManufacturerSelect(advSearchParams['filter_system_manufacturer'].inputElement).then(() => populateModelSelect(advSearchParams['filter_system_model'].inputElement)), renderInventoryTable()]);
					} else {
						await renderInventoryTable();
					}
				} catch (err) {
					console.error(`Error fetching data from filterEl on negation change event listener:`, err);
				}
			});
		}
  
		filterEl.resetElement.addEventListener("click", async (event) => {
			event.preventDefault();
			filterEl.inputElement.value = "";
			if (filterEl.negationElement) filterEl.negationElement.checked = false;
			handleAdvSearchInputChange([filterEl]);
			try {
				if (filterEl.inputElement === filterManufacturer || filterEl.inputElement === filterModel) {
					await Promise.all([populateManufacturerSelect(advSearchParams['filter_system_manufacturer'].inputElement).then(() => populateModelSelect(advSearchParams['filter_system_model'].inputElement)), renderInventoryTable()]);
				} else {
					await renderInventoryTable();
				}
			} catch (err) {
				console.error(`Error fetching data from filterEl on reset event listener:`, err);
			}
		});
	}
}

async function fetchFilteredInventoryData(csvDownload = false): Promise<InventoryTableRow[] | null> {
	const currentParams = new URLSearchParams(window.location.search);

	const apiQuery = new URLSearchParams(currentParams); // API query parameters
	if (currentParams.get('update') === "true") {
		apiQuery.delete("update");
		apiQuery.delete("tagnumber");
		apiQuery.delete("system_serial");
	}

	if (csvDownload) {
		window.location.href = `/api/overview/inventory_table?csv=true&${apiQuery.toString()}`;
		return null;
	}

	try {
		const jsonResponse: InventoryTableRow[] | null = await fetchData(`/api/overview/inventory_table?${apiQuery.toString()}`, false);
		if (jsonResponse === null) console.warn("No data returned from /api/overview/inventory_table");
		return jsonResponse;
	} catch (error) {
		console.warn("Error fetching inventory data:", error);
		return null;
	}
}

async function fetchAllManufacturersAndModels(purgeCache: boolean = false): Promise<Array<AllManufacturersAndModelsRow> | []> {
	const cached = sessionStorage.getItem("uit_manufacturers_and_models");

  try {
		if (cached && !purgeCache) {
			const cacheEntry: ManufacturerAndModelsCache = JSON.parse(cached);
			if (Date.now() - cacheEntry.timestamp < 300000 && Array.isArray(cacheEntry.manufacturers_and_models)) {
				console.log("Loaded manufacturers and models from cache");
				return cacheEntry.manufacturers_and_models;
			}
		}

    const data: AllManufacturersAndModelsRow[] = await fetchData('/api/overview/all_models');
    if (!data || !Array.isArray(data) || data.length === 0) {
      throw new Error('No data returned from /api/overview/all_models');
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

async function populateManufacturerSelect(manufacturerFilterEl: HTMLSelectElement, purgeCache: boolean = false) {
  if (!manufacturerFilterEl) return;

	manufacturerFilterEl.disabled = true;
	manufacturerFilterEl.classList.add('disabled');

	const initialValue = manufacturerFilterEl.value ?? (new URLSearchParams(window.location.search).get('system_manufacturer') || '');
	if (initialValue !== '' && initialValue.trim().length > 0) {
		manufacturerFilterEl.value = initialValue;
		manufacturerFilterEl.disabled = false;
		manufacturerFilterEl.classList.remove('disabled');
	}
	resetSelectElement(manufacturerFilterEl, 'Model', true, undefined);

	try {
  	const data: AllManufacturersAndModelsRow[] = await fetchAllManufacturersAndModels(purgeCache);
		if (!data || !Array.isArray(data) || data.length === 0) throw new Error('No data returned from /api/overview/all_models');

		// Sort manufacturers array - get unique key
		const manufacturerMap = new Map<string, AllManufacturersAndModelsRow>();
		for (const item of data) {
			if (!item.system_manufacturer) continue;
			if (!manufacturerMap.has(item.system_manufacturer)) {
				manufacturerMap.set(item.system_manufacturer, item);
			}
		}
		const uniqueManufacturerArr: AllManufacturersAndModelsRow[] = Array.from(manufacturerMap.values());
		uniqueManufacturerArr.sort((a, b) => {
			const manufacturerA = a.system_manufacturer;
			const manufacturerB = b.system_manufacturer;
			return manufacturerA.localeCompare(manufacturerB);
		});

		// Clear and rebuild manufacturer select options
		resetSelectElement(manufacturerFilterEl, 'Manufacturer', false, undefined);

		// Sort by formatted name
		for (const item of uniqueManufacturerArr) {
			if (!item.system_manufacturer) console.warn("Missing system_manufacturer in uniqueManufacturerArr:", item);
			const option = document.createElement('option');
			option.value = item.system_manufacturer;
			option.textContent = `${item.system_manufacturer} (${item.system_manufacturer_count || 0})`;
			manufacturerFilterEl.appendChild(option);
		}

		const newValue = (initialValue && uniqueManufacturerArr.some(item => item.system_manufacturer === initialValue)) ? initialValue : '';
		manufacturerFilterEl.value = newValue;

		handleAdvSearchInputChange([advSearchParams['filter_system_manufacturer']]);
	} catch (error) {
		console.error('Error fetching manufacturers and models:', error);
	} finally {
		manufacturerFilterEl.disabled = false;
		manufacturerFilterEl.classList.remove('disabled');
	}
}

async function populateModelSelect(modelSelectEl: HTMLSelectElement, purgeCache: boolean = false) {
  if (!modelSelectEl || !modelSelectEl || !filterManufacturer || !filterManufacturerReset) {
		console.warn("Model select element or reset button not found.");
		return;
	};
	
	const initialValue = modelSelectEl.value ? modelSelectEl.value : (new URLSearchParams(window.location.search)).get('system_model') || '';

	if (!filterManufacturer || filterManufacturer.value === '' || filterManufacturer.value.trim().length === 0) {
		// Reset model if no manufacturer is selected
		resetSelectElement(modelSelectEl, 'Model', true);
		handleAdvSearchInputChange([advSearchParams['filter_system_manufacturer'], advSearchParams['filter_system_model']]);
		return;
	}

	try {
		const data: AllManufacturersAndModelsRow[] = await fetchAllManufacturersAndModels(purgeCache);
		if (!data || !Array.isArray(data) || data.length === 0) return;

		data.sort((a, b) => {
			const modelA = a.system_model;
			const modelB = b.system_model;
			return modelA.localeCompare(modelB);
		});

		const filteredData = data.filter(item => item.system_manufacturer === filterManufacturer.value);

		resetSelectElement(modelSelectEl, 'Model');

		for (const item of filteredData) {
			if (!item.system_model) console.warn("Missing system_model in filteredData:", item);
			const option = document.createElement('option');
			option.value = item.system_model;
			option.textContent = item.system_model + ` (${item.system_model_count || 0})`;
			modelSelectEl.appendChild(option);
		}

		const newValue = (initialValue && filteredData.some(item => item.system_model === initialValue)) ? initialValue : '';
		modelSelectEl.value = newValue || '';
		handleAdvSearchInputChange([advSearchParams['filter_system_manufacturer'], advSearchParams['filter_system_model']]);
	} catch (error) {
		console.error('Error fetching manufacturers and models:', error);
		return;
	} finally {
		handleAdvSearchInputChange([advSearchParams['filter_system_manufacturer'], advSearchParams['filter_system_model']]);
		modelSelectEl.disabled = false;
	}
}

async function fetchDomains(purgeCache: boolean = false): Promise<Array<AllDomainsRow> | []> {
	const cached = sessionStorage.getItem("uit_domains");

	try {
		if (cached && !purgeCache) {
			const cacheEntry: DomainCache = JSON.parse(cached);
			if (Date.now() - cacheEntry.timestamp < 300000 && Array.isArray(cacheEntry.domains)) {
				console.log("Loaded domains from cache");
				return cacheEntry.domains;
			}
		}
		const data: Array<AllDomainsRow> = await fetchData('/api/overview/all_domains');
		if (!data || !Array.isArray(data) || data.length === 0) {
			throw new Error('No data returned from /api/overview/all_domains');
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

async function populateDomainSelect(el: HTMLSelectElement, purgeCache: boolean = false) {
	if (!el) return;

	const initialValue = el.value;

	el.disabled = true;

	try {
		const domainData: Array<AllDomainsRow> = await fetchDomains(purgeCache);
		if (!domainData || !Array.isArray(domainData) || domainData.length === 0) {
			throw new Error('No data returned from /api/overview/all_domains');
		}

		domainData.sort((a, b) => {
			if (a.ad_domain && a.ad_domain === 'unknown') return 1;
			if (a.ad_domain && a.ad_domain === 'none') return 2;
			if (b.ad_domain && b.ad_domain === 'unknown') return -1;
			if (b.ad_domain && b.ad_domain === 'none') return -2;
			return (a.ad_domain_formatted || '').localeCompare(b.ad_domain_formatted || '');
		});

		resetSelectElement(el, 'AD Domain', false, undefined);

		for (const domain of domainData) {
			const option = document.createElement('option');
			option.value = domain.ad_domain;
			option.textContent = domain.ad_domain_formatted + (domain.client_count !== null ? ` (${domain.client_count})` : '');
			el.appendChild(option);
		}

		el.value = (initialValue && domainData.some(item => item.ad_domain === initialValue)) ? initialValue : '';
	} catch (error) {
		console.error('Error fetching domains:', error);
	} finally {
		el.disabled = false;
	}
}

async function populateDepartmentSelect(el: HTMLSelectElement, purgeCache: boolean = false) {
	if (!el) return;

	const initialValue = el.value;

	el.disabled = true;

	try {
		const departmentsData: Array<Department> = await fetchDepartments(purgeCache);
		if (!departmentsData || !Array.isArray(departmentsData) || departmentsData.length === 0) {
			throw new Error('No data returned from /api/overview/all_departments');
		}

		resetSelectElement(el, 'Department', false, undefined);

		departmentsData.sort((a, b) => {
			return a.organization_sort_order - b.organization_sort_order;
		});

		for (const department of new Set(departmentsData.map(dep => dep.organization_name_formatted || dep.organization_name))) {
			const orgEl = document.createElement('optgroup');
			orgEl.label = department ? department.trim() : 'N/A';
			el.appendChild(orgEl);
		}

		departmentsData.sort((a, b) => {
			return a.department_name_formatted.localeCompare(b.department_name_formatted);
			// return b.department_sort_order - a.department_sort_order;
		});

		for (const department of departmentsData) {
			const option = document.createElement('option');
			option.value = department.department_name;
			option.textContent = department.department_name_formatted + (department.client_count !== null ? ` (${department.client_count})` : '');
			const parentOptGroup = Array.from(el.getElementsByTagName('optgroup')).find(group => group.label === (department.organization_name_formatted ? department.organization_name_formatted : (department.organization_name ? department.organization_name : 'N/A')));
			if (parentOptGroup) {
				parentOptGroup.appendChild(option);
			} else {
				el.appendChild(option);
			}
		}
		el.value = (initialValue && departmentsData.some(item => initialValue === item.department_name || initialValue === item.department_name_formatted)) ? initialValue : '';
	} catch (error) {
		console.error('Error fetching departments:', error);
	} finally {
		el.disabled = false;
	}
}

async function fetchStatuses(purgeCache: boolean = false): Promise<Record<string, Statuses[]> | null> {
	const cached = sessionStorage.getItem("uit_statuses");

	try {
		if (cached && !purgeCache) {
			const cacheEntry: StatusCache = JSON.parse(cached);
			if (Date.now() - cacheEntry.timestamp < 300000 && cacheEntry.statuses) {
				console.log("Loaded statuses from cache");
				return cacheEntry.statuses;
			}
		}
		const data: Record<string, Statuses[]> = await fetchData('/api/overview/all_statuses');
		if (!data || Object.keys(data).length === 0) {
			throw new Error('No data returned from /api/overview/all_statuses');
		}
		const cacheEntry: StatusCache = {
			timestamp: Date.now(),
			statuses: data
		};
		sessionStorage.setItem("uit_statuses", JSON.stringify(cacheEntry));
		console.log("Cached statuses data");
		return data;
	} catch (error) {
		console.error('Error fetching statuses:', error);
		return null;
	}
}

async function populateStatusSelect(el: HTMLSelectElement, purgeCache: boolean = false) {
	if (!el) return;

	const initialValue = el.value;

	el.disabled = true;

	try {
		const statusMap = await fetchStatuses(purgeCache);
		if (!statusMap || Object.keys(statusMap).length === 0) {
			throw new Error('No data returned from /api/statuses');
		}

		resetSelectElement(el, 'Status', false, undefined);

		let hasInitialValue = false;
		const sortedKeys = Object.keys(statusMap).sort((a, b) => {
			if (a === 'Other') return 1;
			if (b === 'Other') return -1;
			return a.localeCompare(b);
		});

		for (const key of sortedKeys) {
			const statuses = statusMap[key];
			statuses.sort((a, b) => a.status_sort_order - b.status_sort_order);

			const optGroup = document.createElement('optgroup');
			optGroup.label = key;
			el.appendChild(optGroup);

			for (const status of statuses) {
				const option = document.createElement('option');
				option.value = status.status;
				option.textContent = status.status_formatted + (status.client_count !== null ? ` (${status.client_count})` : '');
				optGroup.appendChild(option);

				if (initialValue && (initialValue === status.status || initialValue === status.status_formatted)) {
					hasInitialValue = true;
				}
			}
		}
		
		el.value = hasInitialValue ? initialValue : '';
	} catch (error) {
		console.error('Error fetching statuses:', error);
	} finally {
		el.disabled = false;
	}
}

async function populateLocationSelect(el: HTMLSelectElement, purgeCache: boolean = false) {
	if (!el) return;
	const initialValue = el.value;

	el.disabled = true;
	try {
		const locationData: Array<AllLocations> = await fetchAllLocations(purgeCache);
		if (!locationData || !Array.isArray(locationData) || locationData.length === 0) {
			throw new Error('No data returned from /api/overview/all_locations (populateLocationSelect)');
		}
		locationData.sort((a, b) => { // alpahbetical here, not by timestamp
			const locA = a.location_formatted || a.location || '';
			const locB = b.location_formatted || b.location || '';
			return locA.localeCompare(locB);
		});

		resetSelectElement(el, 'Location', false, undefined);

		for (const location of locationData) {
			const option = document.createElement('option');
			option.value = location.location || '';
			option.textContent = (location.location_formatted || location.location || 'N/A') + 
				(location.location_count !== null ? ` (${location.location_count})` : '');
			el.appendChild(option);
		}

		el.value = (initialValue && locationData.some(item => initialValue === item.location || initialValue === item.location_formatted)) ? initialValue : '';

		el.disabled = false;
	}	catch (error) {
		console.error('Error fetching locations:', error);
	}
}
		

if (inventoryFilterForm) {
	inventoryFilterForm.addEventListener("submit", (event) => {
		event.preventDefault();
		renderInventoryTable();
	});
}

if (advSearchFormReset) {
	advSearchFormReset.addEventListener("click", async (event) => {
		event.preventDefault();

		inventoryTableSearch.value = '';
		inventoryTableSortBy.value = 'time-desc';

		for (const paramName in advSearchParams) {
			const param = advSearchParams[paramName];
			if (!param.inputElement) continue;
			param.inputElement.value = '';
			handleAdvSearchInputChange([param]);
		}

		try {
			if (!(filterDepartment && advSearchLocation && advSearchParams['filter_system_manufacturer'].inputElement && advSearchParams['filter_system_model'].inputElement && filterDomain)) {
				console.warn("One or more filter elements not found, cannot reset filters properly");
				await renderInventoryTable();
				return;
			}
			await Promise.all([
				populateDepartmentSelect(filterDepartment),
				populateLocationSelect(advSearchLocation),
				populateManufacturerSelect(advSearchParams['filter_system_manufacturer'].inputElement).then(() => populateModelSelect(advSearchParams['filter_system_model'].inputElement)),
				populateDomainSelect(filterDomain),
				renderInventoryTable(),
			]);
		} catch (error) {
			console.error("Error resetting filters and fetching data:", error);
		}
	});
}