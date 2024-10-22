
document.addEventListener("DOMContentLoaded", function() {
  // Get the buttons and the navbar element
  const openButton = document.getElementById('open-button');
  const closeButton = document.getElementById('close-button');
  const navbar = document.getElementById('navbar');

  // Log to check if elements exist
  console.log('Open button:', openButton);
  console.log('Close button:', closeButton);
  console.log('Navbar:', navbar);

  // Ensure that buttons and navbar exist
  if (openButton && closeButton && navbar) {
    console.log('Elements found and event listeners ready.');

    // Add event listener to the open button to remove the collapse class
    openButton.addEventListener('click', function() {
      console.log('Open button clicked');
      if (navbar.classList.contains('collapse')) {
        navbar.classList.remove('collapse');
        console.log('Collapse class removed');
      } else {
        console.log('Navbar is already open (no collapse class).');
      }
    });

    // Add event listener to the close button to add the collapse class
    closeButton.addEventListener('click', function() {
      console.log('Close button clicked');
      if (!navbar.classList.contains('collapse')) {
        navbar.classList.add('collapse');
        console.log('Collapse class added');
      } else {
        console.log('Navbar is already collapsed (collapse class exists).');
      }
    });
  } else {
    console.error('Error: Buttons or navbar element not found.');
  }
});