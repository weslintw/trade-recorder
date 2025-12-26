module.exports = {
  root: true,
  extends: ['eslint:recommended'],
  parserOptions: {
    ecmaVersion: 2021,
    sourceType: 'module'
  },
  env: {
    browser: true,
    es2021: true,
    node: true
  },
  plugins: ['svelte'],
  overrides: [
    {
      files: ['*.svelte'],
      processor: 'svelte/svelte'
    }
  ],
  rules: {
    'no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
    'no-console': ['warn', { allow: ['warn', 'error'] }]
  }
};

