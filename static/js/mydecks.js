class DeckManager {
  constructor() {
    this.initElements();
    this.bindEvents();
  }

  initElements() {
    this.showDeckFormBtn = document.getElementById('showDeckForm');
    this.newDeckForm = document.getElementById('newDeckForm');
  }

  bindEvents() {
    // Обработчик кнопки показа/скрытия формы создания колоды
    this.showDeckFormBtn.addEventListener('click', () => this.toggleForm());
  }

  toggleForm() {
    if (this.newDeckForm.style.display === 'none' || this.newDeckForm.style.display === '') {
      this.newDeckForm.style.display = 'block';
      this.showDeckFormBtn.style.display = 'none';
    } else {
      this.newDeckForm.style.display = 'none';
      this.showDeckFormBtn.style.display = 'inline-block';
    }
  }
}

document.addEventListener('DOMContentLoaded', () => {
  new DeckManager();
});
