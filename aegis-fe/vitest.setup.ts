// Vitest setup file
import '@testing-library/jest-dom';

// Mock environment variables
process.env.NEXT_PUBLIC_API_URL = 'http://localhost:8080';

// Mock fetch for all tests
global.fetch = vi.fn();

// Add any global test setup here