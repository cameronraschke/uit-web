document.addEventListener('DOMContentLoaded', () => {
	const printCheckoutFormContainer = document.getElementById('printCheckoutFormContainer');
	const printCheckoutForm = document.getElementById('printCheckoutForm');
	const customerName = document.getElementById('print_customer_name');
	const customerPSID = document.getElementById('print_customer_psid');
	const checkoutDate = document.getElementById('print_checkout_date');
	const returnDate = document.getElementById('print_return_date');
	const confirmAndPrint = document.getElementById('confirmAndPrint');
	

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