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

// Manufacturer filter
const filterManufacturer = document.getElementById('inventory-filter-manufacturer')
const filterManufacturerReset = document.getElementById('inventory-filter-manufacturer-reset')
filterManufacturer.addEventListener("change", async (event) => {
  filterManufacturerReset.style.display = 'inline-block';
  await fetchFilteredInventoryData();
});
filterManufacturerReset.addEventListener("click", async (event) => {
  event.preventDefault();
  filterManufacturerReset.style.display = 'none';
  filterManufacturer.value = '';
  await fetchFilteredInventoryData();
});

// Model filter
const filterModel = document.getElementById('inventory-filter-model')
const filterModelReset = document.getElementById('inventory-filter-model-reset')
filterModel.addEventListener("change", async (event) => {
  filterModelReset.style.display = 'inline-block';
  await fetchFilteredInventoryData();
});
filterModelReset.addEventListener("click", async (event) => {
  event.preventDefault();
  filterModelReset.style.display = 'none';
  filterModel.value = '';
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
