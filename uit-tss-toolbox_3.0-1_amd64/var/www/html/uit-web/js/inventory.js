const inventoryLookupButton = document.getElementById('inventory-lookup-button');
const inventoryLookupForm = document.getElementById('inventory-form-lookup');

function postInventoryData() {
  return null;
}

async function getTagOrSerial(tagnumber, serial) {
  let lookupValue = "";
  if (tagnumber) {
    lookupValue = "?tagnumber=" + tagnumber;
  } else if (serial) {
    lookupValue = "?system_serial=" + serial;
  } else {
    console.log("No tag or serial provided");
    return;
  }
  try {
    const request = await fetchData('/api/lookup' + lookupValue);
    if (!request) {
      throw new Error("Cannot parse json from /api/lookup");
    }

    console.log(request.tagnumber + request.system_serial);

  } catch(e) {
    console.log("Error getting tag/serial:" + e.message);
  }
}

inventoryLookupForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const lookupTag = document.getElementById('inventory-tag-lookup').value;
  const lookupSerial = document.getElementById('inventory-serial-lookup').value;
  await getTagOrSerial(lookupTag, lookupSerial);
});