const inventoryLookupWarningMessage = document.getElementById('existing-inventory-message');
const inventoryLookupTagInput = document.getElementById('inventory-tag-lookup');
const inventoryLookupSerialInput = document.getElementById('inventory-serial-lookup');
const inventoryLookupButton = document.getElementById('inventory-lookup-button');
const inventoryResetButton = document.getElementById('inventory-reset-button');
const inventoryLookupForm = document.getElementById('inventory-form-lookup');

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

  } catch(e) {
    console.log("Error getting tag/serial: " + e.message);
  }
}

inventoryLookupForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const lookupTag = inventoryLookupTagInput.value;
  const lookupSerial = inventoryLookupSerialInput.value;
  const lookupResult = await getTagOrSerial(lookupTag, lookupSerial);
  if (lookupResult) {
    inventoryLookupTagInput.value = lookupResult.tagnumber || "";
    inventoryLookupSerialInput.value = lookupResult.system_serial || "";
    inventoryLookupTagInput.disabled = true;
    inventoryLookupSerialInput.disabled = true;
  } else {
    inventoryLookupWarningMessage.style.display = "block";
    inventoryLookupWarningMessage.textContent = "No inventory found for the provided tag number or serial number. New entry can be created.";
  }
  
  inventoryLookupButton.disabled = true;
  inventoryResetButton.style.display = "inline-block";
});

inventoryResetButton.addEventListener("click", (event) => {
  event.preventDefault();
  inventoryLookupTagInput.value = "";
  inventoryLookupSerialInput.value = "";
  inventoryLookupTagInput.disabled = false;
  inventoryLookupSerialInput.disabled = false;
  inventoryLookupButton.disabled = false;
  inventoryResetButton.style.display = "none";
});