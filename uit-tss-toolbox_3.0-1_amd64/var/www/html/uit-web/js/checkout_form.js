document.addEventListener('DOMContentLoaded', () => {
	const printCheckoutFormContainer = document.getElementById('print-checkout-form-container');
	const printCheckoutForm = document.getElementById('print-checkout-form');
	const customerName = document.getElementById('print-customer-name');
	const customerPSID = document.getElementById('print-customer-psid');
	const checkoutDate = document.getElementById('print-checkout-date');
	const returnDate = document.getElementById('print-return-date');
	const confirmAndPrint = document.getElementById('confirm-and-print');
	

	confirmAndPrint.addEventListener('click', () => {
		document.querySelector('#customer_name').textContent = customerName.value || '';
		document.querySelector('#customer_psid').textContent = customerPSID.value || '';
		document.querySelector('#checkout_date').textContent = checkoutDate.value || '';
		document.querySelector('#return_date').textContent = returnDate.value || '';
		window.print();

		window.addEventListener('afterprint', () => {
			printCheckoutFormContainer.style.display = 'block';
    }, { once: true });
	});
});