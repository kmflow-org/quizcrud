let questionCounter = 0;

function addQuestion() {
    questionCounter++;

    const questionDiv = document.createElement('div');
    questionDiv.className = 'question';
    questionDiv.id = `question-${questionCounter}`;

    questionDiv.innerHTML = `
        <label>Question ${questionCounter}:</label>
        <input type="text" name="question" placeholder="Enter your question here">
        <label>Optional Code Snippet:</label>
        <textarea name="code-snippet" placeholder="Enter code snippet (if any)"></textarea>
        <label>Question Type:</label>
        <select name="question-type" onchange="changeQuestionType(${questionCounter})">
            <option value="single">Single Choice</option>
            <option value="multiple">Multiple Choice</option>
        </select>
        <div class="choices" id="choices-${questionCounter}">
            <div class="choice">
                <input type="radio" class="correct-answer" name="correct-${questionCounter}">
                <input type="text" name="choice" placeholder="Enter choice">
            </div>
        </div>
        <button onclick="addChoice(${questionCounter})">Add Choice</button>
    `;

    document.getElementById('questions-container').appendChild(questionDiv);
}

function changeQuestionType(questionId) {
    const questionType = document.querySelector(`#question-${questionId} select[name="question-type"]`).value;
    const checkboxes = document.querySelectorAll(`#choices-${questionId} .correct-answer`);

    checkboxes.forEach(checkbox => {
        checkbox.type = questionType === 'single' ? 'radio' : 'checkbox';
    });
}

function addChoice(questionId) {
    const questionType = document.querySelector(`#question-${questionId} select[name="question-type"]`).value;
    const choicesDiv = document.getElementById(`choices-${questionId}`);

    const choiceDiv = document.createElement('div');
    choiceDiv.className = 'choice';
    choiceDiv.innerHTML = `
        <input type="${questionType === 'single' ? 'radio' : 'checkbox'}" class="correct-answer" name="correct-${questionId}">
        <input type="text" name="choice" placeholder="Enter choice">
    `;

    choicesDiv.appendChild(choiceDiv);
}

function saveQuiz() {
    const quizTitle = document.getElementById('quiz-title').value;
    if (!quizTitle) {
        alert("Please enter a title for the quiz.");
        return;
    }

    const questions = [];
    const questionsDiv = document.querySelectorAll('.question');

    questionsDiv.forEach(questionDiv => {
        const questionText = questionDiv.querySelector('input[name="question"]').value;
        const codeSnippet = questionDiv.querySelector('textarea[name="code-snippet"]').value;
        const questionType = questionDiv.querySelector('select[name="question-type"]').value;
        const choices = [];
        const correctAnswers = [];

        questionDiv.querySelectorAll('.choice').forEach((choiceDiv, index) => {
            const choiceText = choiceDiv.querySelector('input[name="choice"]').value;
            const isCorrect = choiceDiv.querySelector('.correct-answer').checked;

            choices.push(choiceText);
            if (isCorrect) {
                correctAnswers.push(index);
            }
        });

        questions.push({
            text: questionText,
            code: codeSnippet || null, // Save null if no code snippet is provided
            type: questionType,
            options: choices,
            answers: correctAnswers
        });
    });

    const quiz = {
        title: quizTitle,
        questions: questions
    };

    fetch('http://localhost:8081/create', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(quiz)
    }).then(response => {
        if (response.ok) {
            alert('Quiz saved successfully!');
            window.location.href = '/quizzes';
        } else {
            alert('Failed to save the quiz.');
        }
    }).catch(error => {
        console.error('Error:', error);
        alert('Failed to save the quiz.');
    });
}
