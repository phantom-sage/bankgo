# LoadingAnimation Component

A beautiful, money-themed loading animation component for the BankGo admin dashboard.

## Features

- **SVG-based "BankGo" text outline** for crisp rendering at all sizes
- **Water-filling effect** that animates from bottom to top
- **Money-themed gradient colors**: Gold (#FFD700), Emerald Green (#50C878), Deep Blue (#003366), Silver (#C0C0C0)
- **Realistic water movement** with waves, ripples, and shimmer effects
- **Responsive design** that works across all screen sizes
- **Smooth transition** to dashboard after animation completion
- **Four animation phases**: Initial, Filling, Shimmer, Complete

## Usage

```tsx
import LoadingAnimation from './components/loading/LoadingAnimation';

function App() {
  const handleLoadingComplete = () => {
    // Navigate to dashboard or hide loading screen
    console.log('Loading animation completed');
  };

  return (
    <LoadingAnimation 
      onComplete={handleLoadingComplete}
      duration={3000} // Optional: default is 3000ms
    />
  );
}
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `onComplete` | `() => void` | Required | Callback function called when animation completes |
| `duration` | `number` | `3000` | Total animation duration in milliseconds |

## Animation Phases

1. **Initial (0-500ms)**: Display empty "BankGo" text outline with "Initializing..." text
2. **Filling (500-2500ms)**: Water fills from bottom to top with wave motion and "Loading Dashboard..." text
3. **Shimmer (2500-3000ms)**: Shimmer effect with sparkles and "Almost Ready..." text
4. **Complete (3000ms+)**: Final state with "Welcome to BankGo Admin" before transition

## Visual Effects

- **Background particles**: 20 animated gold particles floating in the background
- **Water gradient**: Multi-color gradient representing money themes
- **Wave pattern**: Animated wave pattern on water surface
- **Ripple effects**: Concentric circles during filling phase
- **Sparkle effects**: 8 animated sparkles during shimmer phase
- **Progress indicator**: Visual progress bar showing animation progress

## Responsive Design

The component uses responsive Tailwind CSS classes to ensure proper display across devices:
- Mobile: `max-w-md`
- Small tablets: `sm:max-w-lg`
- Medium tablets: `md:max-w-xl`
- Large screens: `lg:max-w-2xl`

## Accessibility

- Uses system fonts for better readability
- Animations have reasonable durations (>0.5s) to avoid motion sickness
- Provides meaningful loading text for screen readers
- Respects user preferences for reduced motion (can be enhanced further)

## Performance

- Hardware-accelerated CSS animations
- Optimized SVG rendering
- Minimal DOM manipulation
- 60fps animation targets
- Efficient gradient and pattern definitions

## Testing

The component includes comprehensive tests covering:
- Component rendering and structure
- Animation phases and timing
- Visual effects and responsiveness
- Color themes and accessibility
- Error handling and cleanup

Run tests with:
```bash
npm test -- --testPathPattern=LoadingAnimation.test.tsx
```