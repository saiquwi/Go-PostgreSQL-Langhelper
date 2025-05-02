document.getElementById('wordForm').addEventListener('submit', function(e) {
    const inputs = document.querySelectorAll('.translation-input');
    let filled = false;
    
    inputs.forEach(input => {
        if (input.value.trim() !== '') {
            filled = true;
        }
    });
    
    // Удаляем предыдущее сообщение об ошибке, если оно есть
    const existingError = document.querySelector('.error-message');
    if (existingError) {
        existingError.remove();
    }
    
    if (!filled) {
        e.preventDefault();
        
        // Создаем элемент для сообщения об ошибке
        const errorMessage = document.createElement('div');
        errorMessage.className = 'error-message';
        errorMessage.textContent = 'Please enter at least one translation';
        errorMessage.style.color = 'red';
        errorMessage.style.marginBottom = '10px';
        
        // Находим кнопку и вставляем сообщение перед ней
        const submitButton = document.querySelector('#wordForm button[type="submit"]');
        submitButton.parentNode.insertBefore(errorMessage, submitButton);
        
        // Добавляем визуальное выделение полей
        inputs.forEach(input => {
            input.style.border = '1px solid red';
        });
        return false;
    }
    
    // Сбрасываем стили ошибок, если они были
    inputs.forEach(input => {
        input.style.border = '';
    });
});