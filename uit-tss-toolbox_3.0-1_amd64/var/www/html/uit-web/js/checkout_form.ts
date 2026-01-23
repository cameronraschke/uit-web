document.addEventListener('DOMContentLoaded', () => {
	const printCheckoutFormContainer = document.getElementById('print-checkout-form-container') as HTMLElement;
	const customerName = document.getElementById('print-customer-name') as HTMLInputElement;
	const customerPSID = document.getElementById('print-customer-psid') as HTMLInputElement;
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
		(document.querySelector('#customer_name') as HTMLElement).textContent = customerName.value || '';
		(document.querySelector('#customer_psid') as HTMLElement).textContent = customerPSID.value || '';
		(document.querySelector('#checkout_date') as HTMLElement).textContent = checkoutDateEl.value ? new Date(checkoutDateEl.value + "T00:00:00").toLocaleDateString('en-US', options) : '';
		(document.querySelector('#return_date') as HTMLElement).textContent = returnDateEl.value ? new Date(returnDateEl.value + "T00:00:00").toLocaleDateString('en-US', options) : '';
		window.print();

		window.addEventListener('afterprint', () => {
			printCheckoutFormContainer.style.display = 'block';
    }, { once: true });
	});
});