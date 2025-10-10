let updatingInventory = false;

const inventoryLookupWarningMessage = document.getElementById('existing-inventory-message');
const inventoryLookupForm = document.getElementById('inventory-lookup-form');
const inventoryLookupTagInput = document.getElementById('inventory-tag-lookup');
const inventoryLookupSerialInput = document.getElementById('inventory-serial-lookup');
const inventoryLookupSubmitButton = document.getElementById('inventory-lookup-submit-button');
const inventoryLookupResetButton = document.getElementById('inventory-lookup-reset-button');
const inventoryUpdateForm = document.getElementById('inventory-update-form');
const inventoryUpdateSection = document.getElementById('inventory-update-section');
const inventoryLocationInput = document.getElementById('location');
const inventoryUpdateSubmitButton = document.getElementById('inventory-update-submit-button');
const inventoryUpdateCancelButton = document.getElementById('inventory-update-cancel-button');
const tagDatalist = document.getElementById('inventory-tag-suggestions');

function postInventoryData() {
  return null;
}

async function getTagOrSerial(tagnumber, serial) {
  const query = new URLSearchParams();
  if (tagnumber) {
    query.append("tagnumber", tagnumber);
  } else if (serial) {
    query.append("system_serial", serial);
  } else {
    console.log("No tag or serial provided");
    return;
  }
  try {
    const request = await fetchData(`/api/lookup?${query.toString()}`);
    if (!request) {
      throw new Error("Cannot parse json from /api/lookup");
    }

    const returnObject = {
      tagnumber: request.tagnumber, 
      system_serial: request.system_serial
    }

    return returnObject;

  } catch(error) {
    console.log("Error getting tag/serial: " + error.message);
  }
}

inventoryLookupForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const lookupTag = inventoryLookupTagInput.value;
  const lookupSerial = inventoryLookupSerialInput.value;
  const lookupResult = await getTagOrSerial(lookupTag, lookupSerial);
  if (!lookupTag && !lookupSerial) {
    inventoryLookupWarningMessage.style.display = "block";
    inventoryLookupWarningMessage.textContent = "Please provide a tag number or serial number to look up.";
    return;
  }
  inventoryUpdateSection.style.display = "block";
  if (lookupResult) {
    inventoryLookupTagInput.value = lookupResult.tagnumber || "";
    inventoryLookupSerialInput.value = lookupResult.system_serial || "";
    inventoryLookupTagInput.disabled = true;
    inventoryLookupSerialInput.disabled = true;
    inventoryLookupTagInput.style.backgroundColor = "gainsboro";
    inventoryLookupSerialInput.style.backgroundColor = "gainsboro";
    inventoryLocationInput.focus();
  } else {
    inventoryLookupWarningMessage.style.display = "block";
    inventoryLookupWarningMessage.textContent = "No inventory entry was found for the provided tag number or serial number. A new entry can be created.";
    if (!inventoryLookupTagInput.value) inventoryLookupTagInput.focus();
    else if (!inventoryLookupSerialInput.value) inventoryLookupSerialInput.focus();
  }

  inventoryLookupSubmitButton.disabled = true;
  inventoryLookupSubmitButton.style.cursor = "not-allowed";
  inventoryLookupSubmitButton.style.border = "1px solid gray";
  inventoryLookupResetButton.style.display = "inline-block";
});

function resetInventoryForm() {
  inventoryLookupTagInput.value = "";
  inventoryLookupSerialInput.value = "";
  inventoryLookupTagInput.disabled = false;
  inventoryLookupSerialInput.disabled = false;
  inventoryLookupSubmitButton.disabled = false;
  inventoryLookupSubmitButton.style.cursor = "pointer";
  inventoryLookupSubmitButton.style.border = "1px solid black";
  inventoryLookupResetButton.style.display = "none";
  inventoryLookupForm.reset();
  inventoryUpdateForm.reset();
  inventoryUpdateSection.style.display = "none";
  inventoryLookupWarningMessage.style.display = "none";
  inventoryLookupWarningMessage.textContent = "";
  inventoryLookupTagInput.style.backgroundColor = "initial";
  inventoryLookupSerialInput.style.backgroundColor = "initial";
  inventoryLookupTagInput.focus();
}

inventoryLookupResetButton.addEventListener("click", (event) => {
  event.preventDefault();
  resetInventoryForm();
});

inventoryUpdateCancelButton.addEventListener("click", (event) => {
  event.preventDefault();
  resetInventoryForm();
});

function renderTagOptions(tags) {
  if (!tagDatalist) {
    console.warn("No tag datalist found");
    return;
  }
  tagDatalist.innerHTML = '';
  (tags || []).slice(0, 20).forEach(tag => {
    const option = document.createElement('option');
    option.value = tag;
    tagDatalist.appendChild(option);
  });
}

if (Array.isArray(window.availableTags)) {
  console.log("Available tags found:", window.availableTags);
  renderTagOptions(window.availableTags);
}

document.addEventListener('tags:loaded', (event) => {
  const tags = (event && event.detail && Array.isArray(event.detail.tags)) ? event.detail.tags : window.availableTags;
  renderTagOptions(tags || []);
});

inventoryLookupTagInput.addEventListener("keyup", (event) => {
  const searchTerm = (event.target.value || '').trim().toLowerCase();
  const allTags = Array.isArray(window.availableTags) ? window.availableTags : [];
  const filteredTags = searchTerm
    ? allTags.filter(tag => String(tag).trim().includes(searchTerm))
    : allTags;
  if (filteredTags.includes(searchTerm)) {
    tagDatalist.innerHTML = '';
  } else {
    renderTagOptions(filteredTags);
  }
});

inventoryUpdateForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  inventoryUpdateSubmitButton.disabled = true;
  if (updatingInventory) return;
  updatingInventory = true;

  const formData = new FormData();
  if (inventoryLookupTagInput && inventoryLookupSerialInput) {
    inventoryLookupForm.querySelector("#inventory-tag-lookup").value ? formData.append("tagnumber", Number(inventoryLookupForm.querySelector("#inventory-tag-lookup").value)) : formData.append("tagnumber", null);
    inventoryLookupForm.querySelector("#inventory-serial-lookup").value ? formData.append("system_serial", String(inventoryLookupForm.querySelector("#inventory-serial-lookup").value)) : formData.append("system_serial", null);
    inventoryUpdateForm.querySelector("#location").value ? formData.append("location", String(inventoryUpdateForm.querySelector("#location").value)) : formData.append("location", null);
    inventoryUpdateForm.querySelector("#system_manufacturer").value ? formData.append("system_manufacturer", String(inventoryUpdateForm.querySelector("#system_manufacturer").value)) : formData.append("system_manufacturer", null);
    inventoryUpdateForm.querySelector("#system_model").value ? formData.append("system_model", String(inventoryUpdateForm.querySelector("#system_model").value)) : formData.append("system_model", null);
    inventoryUpdateForm.querySelector("#department").value ? formData.append("department", String(inventoryUpdateForm.querySelector("#department").value)) : formData.append("department", null);
    inventoryUpdateForm.querySelector("#domain").value ? formData.append("domain", String(inventoryUpdateForm.querySelector("#domain").value)) : formData.append("domain", null);
    inventoryUpdateForm.querySelector("#working").value ? formData.append("working", Boolean(inventoryUpdateForm.querySelector("#working").value)) : formData.append("working", null);
    inventoryUpdateForm.querySelector("#status").value ? formData.append("status", String(inventoryUpdateForm.querySelector("#status").value)) : formData.append("status", null);
    inventoryUpdateForm.querySelector("#note").value ? formData.append("note", String(inventoryUpdateForm.querySelector("#note").value)) : formData.append("note", null);
    var fileCount = 0;
    for (const file of inventoryUpdateForm.querySelector("#inventory-file-input").files) {
      fileCount++;
      formData.append("image-" + fileCount, file);
    }
  } else {
    throw new Error("No tag or serial input fields found in DOM");
  }
  const jsonData = Object.fromEntries(formData.entries());

  try {
    const response = await fetch("/api/update_inventory", {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify(jsonData)
    });

    const body = await response.text();
    console.log(body);

    if (!response.ok) {
      throw new Error("Failed to update inventory");
    }

    const result = await response.json();
    console.log("Inventory updated successfully:", result);
  } catch (error) {
    console.error("Error updating inventory:", error);
  } finally {
    inventoryUpdateSubmitButton.disabled = false;
  }
});
