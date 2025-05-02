document.addEventListener("DOMContentLoaded", function() {
    const menuToggle = document.getElementById('menu-toggle');
    const sidebar = document.getElementById('sidebar');

    // Управление основным меню (сайдбаром)
    if (menuToggle && sidebar) {
        menuToggle.addEventListener('click', () => {
            sidebar.classList.toggle('active');
            sidebar.style.left = sidebar.classList.contains('active') ? '0' : '-200px';
        });
    }

    // Управление подменю
    const submenuToggles = document.querySelectorAll('.submenu-toggle');

    submenuToggles.forEach(toggle => {
        toggle.addEventListener('click', function(event) {
            event.preventDefault(); // Предотвращаем переход по ссылке
            // Закрываем все подменю
            const allSubmenus = document.querySelectorAll('.submenu');
            allSubmenus.forEach(submenu => {
                submenu.style.display = "none";
            });

            // Открываем только текущее подменю
            const submenu = this.nextElementSibling; // Получаем следующее подменю
            if (submenu) {
                submenu.style.display = submenu.style.display === "block" ? "none" : "block"; // Переключаем состояние
            }
        });
    });
});