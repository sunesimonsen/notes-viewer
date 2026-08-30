document.addEventListener("keydown", (event) => {
  if (event.key === "k" && (event.metaKey || event.ctrlKey)) {
    event.preventDefault();

    const dialog = document.getElementById("search-dialog");
    const input = document.getElementById("search");
    if (!dialog || !input) return;

    if (!dialog.open) {
      dialog.showModal();
    }
    input.focus();
  }
});
