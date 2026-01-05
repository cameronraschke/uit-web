document.addEventListener('DOMContentLoaded', () => {
	const printCheckoutFormContainer = document.getElementById('print-checkout-form-container') as HTMLElement;
	const customerName = document.getElementById('print-customer-name') as HTMLInputElement;
	const customerPSID = document.getElementById('print-customer-psid') as HTMLInputElement;
	const checkoutDate = document.getElementById('print-checkout-date') as HTMLInputElement;
	const returnDate = document.getElementById('print-return-date') as HTMLInputElement;
	const confirmAndPrint = document.getElementById('confirm-and-print') as HTMLElement;
	

	confirmAndPrint.addEventListener('click', () => {
		(document.querySelector('#customer_name') as HTMLElement).textContent = customerName.value || '';
		(document.querySelector('#customer_psid') as HTMLElement).textContent = customerPSID.value || '';
		(document.querySelector('#checkout_date') as HTMLElement).textContent = checkoutDate.value || '';
		(document.querySelector('#return_date') as HTMLElement).textContent = returnDate.value || '';
		window.print();

		window.addEventListener('afterprint', () => {
			printCheckoutFormContainer.style.display = 'block';
    }, { once: true });
	});
});