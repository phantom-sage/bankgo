import React, { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { LoadingAnimationProps } from '../../types';

const LoadingAnimation: React.FC<LoadingAnimationProps> = ({
  onComplete,
  duration = 3000,
}) => {
  const [animationPhase, setAnimationPhase] = useState<
    'initial' | 'filling' | 'shimmer' | 'complete'
  >('initial');

  useEffect(() => {
    // Phase 1: Initial display (0-500ms)
    const initialTimer = setTimeout(() => {
      setAnimationPhase('filling');
    }, 500);

    // Phase 2: Water filling (500-2500ms)
    const fillingTimer = setTimeout(() => {
      setAnimationPhase('shimmer');
    }, 2500);

    // Phase 3: Shimmer effect (2500-3000ms)
    const shimmerTimer = setTimeout(() => {
      setAnimationPhase('complete');
    }, duration);

    // Phase 4: Transition to dashboard
    const completeTimer = setTimeout(() => {
      onComplete();
    }, duration + 500);

    return () => {
      clearTimeout(initialTimer);
      clearTimeout(fillingTimer);
      clearTimeout(shimmerTimer);
      clearTimeout(completeTimer);
    };
  }, [onComplete, duration]);

  return (
    <motion.div
      className="fixed inset-0 bg-gradient-to-br from-deepBlue via-admin-primary to-deepBlue flex items-center justify-center overflow-hidden"
      initial={{ opacity: 1 }}
      animate={{ opacity: animationPhase === 'complete' ? 0 : 1 }}
      transition={{
        duration: 0.5,
        delay: animationPhase === 'complete' ? 0 : 0,
      }}
    >
      {/* Background animated particles */}
      <div className="absolute inset-0 overflow-hidden">
        {[...Array(20)].map((_, i) => (
          <motion.div
            key={i}
            className="absolute w-2 h-2 bg-gold/20 rounded-full"
            style={{
              left: `${Math.random() * 100}%`,
              top: `${Math.random() * 100}%`,
            }}
            animate={{
              y: [0, -20, 0],
              opacity: [0.2, 0.8, 0.2],
              scale: [0.5, 1, 0.5],
            }}
            transition={{
              duration: 3 + Math.random() * 2,
              repeat: Infinity,
              delay: Math.random() * 2,
            }}
          />
        ))}
      </div>

      {/* Main animation container */}
      <div className="relative z-10 flex items-center justify-center">
        <svg
          width="400"
          height="120"
          viewBox="0 0 400 120"
          className="w-full max-w-md sm:max-w-lg md:max-w-xl lg:max-w-2xl h-auto"
          style={{ filter: 'drop-shadow(0 4px 8px rgba(0,0,0,0.3))' }}
        >
          {/* Gradient definitions */}
          <defs>
            {/* Money-themed gradient for water */}
            <linearGradient
              id="waterGradient"
              x1="0%"
              y1="100%"
              x2="0%"
              y2="0%"
            >
              <stop offset="0%" stopColor="#FFD700" stopOpacity="1" />
              <stop offset="25%" stopColor="#50C878" stopOpacity="0.9" />
              <stop offset="50%" stopColor="#003366" stopOpacity="0.8" />
              <stop offset="75%" stopColor="#C0C0C0" stopOpacity="0.9" />
              <stop offset="100%" stopColor="#FFD700" stopOpacity="1" />
            </linearGradient>

            {/* Shimmer gradient */}
            <linearGradient
              id="shimmerGradient"
              x1="0%"
              y1="0%"
              x2="100%"
              y2="0%"
            >
              <stop offset="0%" stopColor="rgba(255,255,255,0)" />
              <stop offset="50%" stopColor="rgba(255,255,255,0.8)" />
              <stop offset="100%" stopColor="rgba(255,255,255,0)" />
              <animateTransform
                attributeName="gradientTransform"
                type="translate"
                values="-100 0;400 0;-100 0"
                dur="1.5s"
                repeatCount="indefinite"
              />
            </linearGradient>

            {/* Text outline stroke */}
            <linearGradient
              id="strokeGradient"
              x1="0%"
              y1="0%"
              x2="100%"
              y2="0%"
            >
              <stop offset="0%" stopColor="#C0C0C0" />
              <stop offset="50%" stopColor="#FFD700" />
              <stop offset="100%" stopColor="#C0C0C0" />
            </linearGradient>

            {/* Clipping path for water effect */}
            <clipPath id="textClip">
              <text
                x="200"
                y="75"
                textAnchor="middle"
                dominantBaseline="middle"
                fontSize="48"
                fontWeight="bold"
                fontFamily="system-ui, -apple-system, sans-serif"
              >
                BankGo
              </text>
            </clipPath>

            {/* Wave pattern for water surface */}
            <pattern
              id="wavePattern"
              x="0"
              y="0"
              width="40"
              height="8"
              patternUnits="userSpaceOnUse"
            >
              <path
                d="M0,4 Q10,0 20,4 T40,4"
                stroke="rgba(255,255,255,0.3)"
                strokeWidth="1"
                fill="none"
              >
                <animateTransform
                  attributeName="transform"
                  type="translate"
                  values="0,0;-40,0;0,0"
                  dur="2s"
                  repeatCount="indefinite"
                />
              </path>
            </pattern>
          </defs>

          {/* Text outline */}
          <text
            x="200"
            y="75"
            textAnchor="middle"
            dominantBaseline="middle"
            fontSize="48"
            fontWeight="bold"
            fontFamily="system-ui, -apple-system, sans-serif"
            fill="none"
            stroke="url(#strokeGradient)"
            strokeWidth="2"
            opacity={animationPhase === 'initial' ? 1 : 0.3}
          >
            BankGo
          </text>

          {/* Water filling effect */}
          <g clipPath="url(#textClip)">
            <motion.rect
              x="0"
              y="120"
              width="400"
              height="120"
              fill="url(#waterGradient)"
              initial={{ y: 120 }}
              animate={{
                y:
                  animationPhase === 'filling' ||
                  animationPhase === 'shimmer' ||
                  animationPhase === 'complete'
                    ? 0
                    : 120,
              }}
              transition={{
                duration: 2,
                ease: 'easeInOut',
                delay: 0.5,
              }}
            />

            {/* Water surface waves */}
            {(animationPhase === 'filling' || animationPhase === 'shimmer') && (
              <motion.rect
                x="0"
                y="0"
                width="400"
                height="8"
                fill="url(#wavePattern)"
                initial={{ y: 120 }}
                animate={{ y: 0 }}
                transition={{
                  duration: 2,
                  ease: 'easeInOut',
                  delay: 0.5,
                }}
              />
            )}

            {/* Ripple effects */}
            {animationPhase === 'filling' && (
              <>
                {[...Array(3)].map((_, i) => (
                  <motion.circle
                    key={i}
                    cx="200"
                    cy="60"
                    r="0"
                    fill="none"
                    stroke="rgba(255,255,255,0.4)"
                    strokeWidth="2"
                    initial={{ r: 0, opacity: 0 }}
                    animate={{
                      r: [0, 30, 60],
                      opacity: [0, 0.6, 0],
                    }}
                    transition={{
                      duration: 1.5,
                      repeat: Infinity,
                      delay: i * 0.5 + 1,
                    }}
                  />
                ))}
              </>
            )}

            {/* Shimmer overlay */}
            {animationPhase === 'shimmer' && (
              <rect
                x="0"
                y="0"
                width="400"
                height="120"
                fill="url(#shimmerGradient)"
              />
            )}
          </g>

          {/* Sparkle effects */}
          {(animationPhase === 'shimmer' || animationPhase === 'complete') && (
            <>
              {[...Array(8)].map((_, i) => (
                <motion.g key={i}>
                  <motion.path
                    d="M0,-4 L1,0 L0,4 L-1,0 Z"
                    fill="#FFD700"
                    transform={`translate(${50 + i * 40}, ${
                      30 + (i % 2) * 20
                    })`}
                    initial={{ scale: 0, rotate: 0 }}
                    animate={{
                      scale: [0, 1, 0],
                      rotate: [0, 180, 360],
                    }}
                    transition={{
                      duration: 1,
                      repeat: Infinity,
                      delay: i * 0.2 + 2.5,
                    }}
                  />
                </motion.g>
              ))}
            </>
          )}
        </svg>
      </div>

      {/* Loading text */}
      <motion.div
        className="absolute bottom-20 left-1/2 transform -translate-x-1/2 text-center"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 1, duration: 0.5 }}
      >
        <div className="text-white/80 text-sm font-medium tracking-wide">
          {animationPhase === 'initial' && 'Initializing...'}
          {animationPhase === 'filling' && 'Loading Dashboard...'}
          {animationPhase === 'shimmer' && 'Almost Ready...'}
          {animationPhase === 'complete' && 'Welcome to BankGo Admin'}
        </div>

        {/* Progress indicator */}
        <motion.div className="mt-3 w-48 h-1 bg-white/20 rounded-full overflow-hidden mx-auto">
          <motion.div
            className="h-full bg-gradient-to-r from-gold via-emerald to-silver rounded-full"
            initial={{ width: '0%' }}
            animate={{
              width:
                animationPhase === 'initial'
                  ? '10%'
                  : animationPhase === 'filling'
                    ? '70%'
                    : animationPhase === 'shimmer'
                      ? '90%'
                      : '100%',
            }}
            transition={{ duration: 0.5, ease: 'easeInOut' }}
          />
        </motion.div>
      </motion.div>
    </motion.div>
  );
};

export default LoadingAnimation;
