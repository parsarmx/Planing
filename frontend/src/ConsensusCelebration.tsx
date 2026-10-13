import { useEffect, useRef } from 'react';
import { PartyPopper } from 'lucide-react';

const duration = 3600;
const colors = ['#b69af4', '#ebaccc', '#87c8e3', '#a1d9bc', '#edc878', '#eeab8b'];

type Particle = {
  side: number;
  start: number;
  speed: number;
  lift: number;
  delay: number;
  angle: number;
  spin: number;
  size: number;
  shape: number;
  color: string;
};

export function ConsensusCelebration({ value, onDone }: { value: string; onDone: () => void }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const doneRef = useRef(onDone);
  doneRef.current = onDone;

  useEffect(() => {
    const canvas = canvasRef.current;
    const context = canvas?.getContext('2d');
    const motion = window.matchMedia('(prefers-reduced-motion: reduce)');
    if (!canvas || !context || motion.matches) {
      const timeout = window.setTimeout(() => doneRef.current(), 2400);
      return () => window.clearTimeout(timeout);
    }

    let width = 0;
    let height = 0;
    let frame = 0;
    let finished = false;
    const resize = () => {
      width = window.innerWidth;
      height = window.innerHeight;
      const ratio = Math.min(window.devicePixelRatio || 1, 2);
      canvas.width = Math.round(width * ratio);
      canvas.height = Math.round(height * ratio);
      context.setTransform(ratio, 0, 0, ratio, 0, 0);
    };
    resize();

    const particles: Particle[] = Array.from({ length: width < 600 ? 90 : 150 }, (_, i) => ({
      side: i % 2 ? 1 : -1,
      start: 0.1 + Math.random() * 0.12,
      speed: 0.22 + Math.random() * 0.48,
      lift: 0.65 + Math.random() * 0.55,
      delay: Math.random() * 180 + (i % 3 === 0 ? 220 : 0),
      angle: Math.random() * Math.PI,
      spin: (Math.random() - 0.5) * 9,
      size: 4 + Math.random() * 5,
      shape: i % 4,
      color: colors[i % colors.length],
    }));
    const start = performance.now();
    const finish = () => {
      if (finished) return;
      finished = true;
      window.cancelAnimationFrame(frame);
      doneRef.current();
    };
    const visibility = () => {
      if (document.hidden) finish();
    };

    const render = (now: number) => {
      if (finished) return;
      const elapsed = now - start;
      context.clearRect(0, 0, width, height);
      if (elapsed >= duration) {
        finish();
        return;
      }

      for (const particle of particles) {
        const time = (elapsed - particle.delay) / 1000;
        if (time < 0) continue;
        const drag = (1 - Math.exp(-time * 0.8)) / 0.8;
        const origin = particle.side === 1 ? width * particle.start : width * (1 - particle.start);
        const x = origin + particle.side * width * particle.speed * drag + Math.sin(time * 4) * 9;
        const y =
          height * 0.82 - Math.min(height * 1.25, 1100) * particle.lift * time + 360 * time * time;
        if (y > height + 30) continue;

        context.save();
        context.globalAlpha = Math.min(1, time / 0.12, (duration - elapsed) / 700);
        context.fillStyle = particle.color;
        context.translate(x, y);
        context.rotate(particle.angle + particle.spin * time);
        context.scale(Math.max(0.25, Math.abs(Math.cos(time * 5 + particle.angle))), 1);
        if (particle.shape === 0) {
          context.beginPath();
          context.arc(0, 0, particle.size * 0.55, 0, Math.PI * 2);
          context.fill();
        } else {
          const length = particle.shape === 1 ? particle.size * 2.6 : particle.size * 1.4;
          context.beginPath();
          context.roundRect(-particle.size / 2, -length / 2, particle.size, length, 2);
          context.fill();
        }
        context.restore();
      }
      frame = window.requestAnimationFrame(render);
    };

    if (document.hidden) finish();
    else frame = window.requestAnimationFrame(render);
    window.addEventListener('resize', resize);
    document.addEventListener('visibilitychange', visibility);
    motion.addEventListener('change', finish);
    return () => {
      finished = true;
      window.cancelAnimationFrame(frame);
      window.removeEventListener('resize', resize);
      document.removeEventListener('visibilitychange', visibility);
      motion.removeEventListener('change', finish);
    };
  }, []);

  const score = Number.isFinite(Number(value)) ? Number(value).toLocaleString('fa-IR') : value;
  return (
    <div className="consensus-celebration">
      <canvas ref={canvasRef} className="consensus-confetti" aria-hidden="true" />
      <div className="consensus-toast-position">
        <div className="consensus-toast" role="status" aria-live="polite">
          <span className="consensus-toast-icon">
            <PartyPopper size={25} />
          </span>
          <div>
            <strong>همه هم‌نظر هستید!</strong>
            <span>یک تیم، یک تخمین مشترک</span>
          </div>
          <b className="consensus-toast-score">{score}</b>
        </div>
      </div>
    </div>
  );
}
