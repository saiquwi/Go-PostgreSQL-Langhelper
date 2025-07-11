document.addEventListener("DOMContentLoaded", function () {
  // Показать форму добавления
  document.getElementById("showFormBtn").addEventListener("click", function () {
    document.getElementById("addWordForm").style.display = "block";
    document.getElementById("formTitle").textContent = "Add New Word";
    document.getElementById("wordIdField").value = "";
    document.querySelectorAll(".translation-input").forEach(input => input.value = "");
  });

  // Отмена редактирования/добавления
  document.getElementById("cancelBtn").addEventListener("click", function () {
    document.getElementById("addWordForm").style.display = "none";
    document.getElementById("formTitle").textContent = "Add New Word";
    document.getElementById("wordIdField").value = "";
    document.querySelectorAll(".translation-input").forEach(input => input.value = "");
  });

  // Редактирование существующего слова
  document.querySelectorAll(".edit-btn").forEach(button => {
    button.addEventListener("click", function () {
      const wordId = this.dataset.wordId;
      const row = this.closest("tr");

      document.getElementById("formTitle").textContent = "Edit Word";
      document.getElementById("wordIdField").value = wordId;

      // Очистка формы
      document.querySelectorAll(".translation-input").forEach(input => input.value = "");

      // Заполнить форму переводами из таблицы
      row.querySelectorAll("td[data-lang-id]").forEach(cell => {
        const langId = cell.dataset.langId;
        const translation = cell.dataset.translation;

        const input = document.querySelector(`input[name='translation_${langId}']`);
        if (input) input.value = translation;
      });

      document.getElementById("addWordForm").style.display = "block";
    });
  });

  // Удаление слова с подтверждением
  document.querySelectorAll(".delete-btn").forEach(button => {
    button.addEventListener("click", function () {
      const wordId = this.dataset.wordId;
      if (confirm("Are you sure you want to delete this word?")) {
        window.location.href = `/mywords/delete/${wordId}`;
      }
    });
  });
});
