document.addEventListener('DOMContentLoaded', () => {
	const printCheckoutFormContainer = document.getElementById('print-checkout-form-container') as HTMLElement;
	const customerNameEl = document.getElementById('print-customer-name') as HTMLInputElement;
	const customerPSIDEl = document.getElementById('print-customer-psid') as HTMLInputElement;
	const checkoutDateEl = document.getElementById('print-checkout-date') as HTMLInputElement;
	const returnDateEl = document.getElementById('print-return-date') as HTMLInputElement;
	const confirmAndPrint = document.getElementById('confirm-and-print') as HTMLElement;
	
	const options: Intl.DateTimeFormatOptions = {
		timeZone: "America/Chicago",
		year: "numeric",
		month: "long",
		weekday: "long",
		day: "numeric",
	};

	confirmAndPrint.addEventListener('click', () => {

		const customerName = customerNameEl.value.trim() || '';
		const customerPSID = customerPSIDEl.value.trim() || '';
		const checkoutDateStr = checkoutDateEl.value.trim() || '';
		const returnDateStr = returnDateEl.value.trim() || '';


		(document.querySelector('#customer_name') as HTMLElement).textContent = customerName;
		(document.querySelector('#customer_psid') as HTMLElement).textContent = customerPSID;
		(document.querySelector('#checkout_date') as HTMLElement).textContent =  checkoutDateStr ? new Date(checkoutDateStr + "T00:00:00").toLocaleDateString('en-US', options) : '';
		(document.querySelector('#return_date') as HTMLElement).textContent = returnDateStr ? new Date(returnDateStr + "T00:00:00").toLocaleDateString('en-US', options) : '';

		if (customerName || customerPSID) {
			const urlParams = new URLSearchParams(window.location.search)
			let fileName = urlParams.get("tagnumber") + " Checkout Form";
			if (customerName) fileName = fileName + ` - ${customerName}`;
			if (customerPSID) fileName = fileName + ` (${customerPSID})`;
			window.document.title = fileName;
		}

		window.print();

		window.addEventListener('afterprint', () => {
			printCheckoutFormContainer.style.display = 'block';
    }, { once: true });
	});
});