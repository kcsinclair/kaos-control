# prompts used

## Initial prompt
```
I have written some high level requirements in high-level-requirements.md for an idea I have had, grill me on this document so you can write more detailed requirements.
```

### notes

high-level-requirements.md was really the idea!

The process of going from high level requirements (idea) helped me to flesh out the idea and guide the technical direction and identify any architectural changesd neede.

## going meta

Going to restructure the directory now and create initialise for CLAUDE.md

requirements-questions.md renamed to lifecycle/requirements/Innovation Maker - Making Releases from Ideas-questions.md
detailed-requirements.md renamed to lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md

## second prompt for the plans

Opus should read the requirements and create plans for backend, frontend and test

```
Please read the detailed requirements in the document "lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md".  Using these requirements you will create the development plans for other agents, the result should be three files
- "lifecycle/backend-plans/Innovation Maker - Making Releases from Ideas-2-be.md"
- "lifecycle/frontend-plans/Innovation Maker - Making Releases from Ideas-3-fe.md"
- "lifecycle/test-plans/Innovation Maker - Making Releases from Ideas-4-test.md"
Please let me know if you have any questions.
```

## development time

Then going to use a sonnet agent to start creating the code. 

### third prompt for the coding backend

### fourth prompt for the coding frontend

### fifth prompt for the coding tests

