import { useEffect, useState, type CSSProperties } from 'react';
import { Check, Shuffle } from 'lucide-react';

export type AvatarConfig = {
  version: 1;
  face: string;
  skin: string;
  hair: string;
  hair_color: string;
  expression: string;
  glasses: string;
  shirt: string;
};
type Choice = { value: string; label: string; color?: string };
type Part = Exclude<keyof AvatarConfig, 'version'> | 'background';
const skins: Record<string, string> = {
  porcelain: '#ffe0ce',
  warm: '#f7c59b',
  sand: '#e7ab77',
  tan: '#bd8056',
  deep: '#925a3e',
  rich: '#633f34',
};
const hairColors: Record<string, string> = {
  espresso: '#30282e',
  brown: '#694137',
  gold: '#e8b34b',
  copper: '#b95136',
  silver: '#c5ccd8',
  pink: '#df6f9e',
};
const shirts: Record<string, string> = {
  violet: '#8360da',
  pink: '#df5b98',
  mint: '#45ac91',
  blue: '#4f94d2',
  amber: '#e7ae3f',
  coral: '#e77860',
};
export const backgroundChoices: Choice[] = [
  { value: 'violet', label: 'بنفش', color: '#ede5fc' },
  { value: 'pink', label: 'صورتی', color: '#f9e2ed' },
  { value: 'mint', label: 'نعنایی', color: '#dff2e9' },
  { value: 'blue', label: 'آبی', color: '#e2edfc' },
  { value: 'peach', label: 'هلویی', color: '#fbe9dd' },
  { value: 'yellow', label: 'زرد', color: '#f8efd5' },
];
const labels: Record<string, string> = {
  porcelain: 'روشن',
  warm: 'گندمی روشن',
  sand: 'گندمی',
  tan: 'برنزه',
  deep: 'تیره',
  rich: 'تیره عمیق',
  espresso: 'مشکی',
  brown: 'قهوه‌ای',
  gold: 'طلایی',
  copper: 'مسی',
  silver: 'نقره‌ای',
  pink: 'صورتی',
  violet: 'بنفش',
  mint: 'نعنایی',
  blue: 'آبی',
  amber: 'کهربایی',
  coral: 'مرجانی',
};
const colorChoices = (colors: Record<string, string>): Choice[] =>
  Object.entries(colors).map(([value, color]) => ({ value, color, label: labels[value] }));
const parts: { id: Part; label: string; choices: Choice[] }[] = [
  {
    id: 'face',
    label: 'فرم چهره',
    choices: [
      { value: 'round', label: 'گرد' },
      { value: 'square', label: 'چهارگوش' },
      { value: 'oval', label: 'کشیده' },
    ],
  },
  { id: 'skin', label: 'رنگ پوست', choices: colorChoices(skins) },
  {
    id: 'hair',
    label: 'مدل مو',
    choices: [
      { value: 'crop', label: 'کوتاه' },
      { value: 'swoop', label: 'یک‌طرفه' },
      { value: 'bob', label: 'کاره' },
      { value: 'curls', label: 'فر' },
      { value: 'bun', label: 'گوجه‌ای' },
      { value: 'bald', label: 'بدون مو' },
      { value: 'scarf', label: 'روسری' },
    ],
  },
  { id: 'hair_color', label: 'رنگ مو', choices: colorChoices(hairColors) },
  {
    id: 'expression',
    label: 'حالت چهره',
    choices: [
      { value: 'smile', label: 'لبخند' },
      { value: 'laugh', label: 'خنده' },
      { value: 'wink', label: 'چشمک' },
      { value: 'calm', label: 'آرام' },
    ],
  },
  {
    id: 'glasses',
    label: 'عینک',
    choices: [
      { value: 'none', label: 'بدون عینک' },
      { value: 'round', label: 'گرد' },
      { value: 'square', label: 'چهارگوش' },
    ],
  },
  { id: 'shirt', label: 'رنگ لباس', choices: colorChoices(shirts) },
  { id: 'background', label: 'پس‌زمینه', choices: backgroundChoices },
];

export function avatarFor(seed: string): AvatarConfig {
  let hash = 0;
  for (const char of seed) hash = (Math.imul(hash, 31) + char.codePointAt(0)!) >>> 0;
  return {
    version: 1,
    face: ['round', 'square', 'oval'][hash % 3],
    skin: Object.keys(skins)[(hash >>> 3) % 6],
    hair: ['crop', 'swoop', 'bob', 'curls', 'bun'][(hash >>> 5) % 5],
    hair_color: ['espresso', 'brown', 'gold', 'copper'][(hash >>> 8) % 4],
    expression: 'smile',
    glasses: hash % 4 === 0 ? 'round' : 'none',
    shirt: Object.keys(shirts)[(hash >>> 11) % 6],
  };
}

// Original, composable vector artwork; all parts stay sharp at thumbnail size.
export function AvatarArt({ config: c }: { config: AvatarConfig }) {
  const skin = skins[c.skin] || skins.warm;
  const hair = hairColors[c.hair_color] || hairColors.espresso;
  const shirt = shirts[c.shirt] || shirts.violet;
  const ink = '#38272d';
  const darkSkin = ['deep', 'rich'].includes(c.skin);
  return (
    <svg className="avatar-art" viewBox="0 0 120 120" aria-hidden="true" focusable="false">
      {(c.hair === 'bob' || c.hair === 'scarf') && (
        <rect x="23" y="17" width="74" height="94" rx="34" fill={hair} />
      )}
      {c.hair === 'curls' && (
        <g fill={hair}>
          <circle cx="37" cy="33" r="18" />
          <circle cx="58" cy="24" r="19" />
          <circle cx="80" cy="34" r="18" />
          <rect x="21" y="36" width="78" height="65" rx="22" />
        </g>
      )}
      {c.hair === 'bun' && <circle cx="60" cy="17" r="17" fill={hair} />}
      <path d="M13 120v-7c0-12 17-20 35-20h24c18 0 35 8 35 20v7" fill={shirt} />
      <path d="M48 95l12 15 12-15" fill="#fff" opacity=".2" />
      <rect x="51" y="78" width="18" height="24" rx="9" fill={skin} />
      <path d="M52 86h16v6c-5 4-11 3-16 0" fill={ink} opacity=".09" />
      {c.hair !== 'scarf' && (
        <g fill={skin}>
          <circle cx="31" cy="61" r="8" />
          <circle cx="89" cy="61" r="8" />
        </g>
      )}
      {c.face === 'square' ? (
        <rect x="31" y="27" width="58" height="62" rx="15" fill={skin} />
      ) : c.face === 'oval' ? (
        <rect x="35" y="22" width="50" height="70" rx="25" fill={skin} />
      ) : (
        <rect x="29" y="27" width="62" height="63" rx="28" fill={skin} />
      )}
      <g fill="#df826d" opacity={darkSkin ? '.22' : '.25'}>
        <ellipse cx="39" cy="68" rx="8" ry="5" />
        <ellipse cx="81" cy="68" rx="8" ry="5" />
      </g>
      <g stroke={ink} strokeWidth="3" strokeLinecap="round" fill="none">
        <path d="M41 41q7-4 13 0" />
        <path d="M67 41q7-4 13 0" />
      </g>
      {c.expression === 'laugh' ? (
        <g stroke={ink} strokeWidth="4" strokeLinecap="round" fill="none">
          <path d="M41 55q7-8 14 0" />
          <path d="M65 55q7-8 14 0" />
        </g>
      ) : (
        <>
          <rect x="40" y="45" width="17" height="20" rx="8" fill="#fffdf5" />
          <rect x="47" y="49" width="7" height="12" rx="3.5" fill={ink} />
          <circle cx="50" cy="51" r="1.4" fill="white" />
          {c.expression === 'wink' ? (
            <path
              d="M65 55q7-7 14 0"
              fill="none"
              stroke={ink}
              strokeWidth="4"
              strokeLinecap="round"
            />
          ) : (
            <>
              <rect x="63" y="45" width="17" height="20" rx="8" fill="#fffdf5" />
              <rect x="70" y="49" width="7" height="12" rx="3.5" fill={ink} />
              <circle cx="73" cy="51" r="1.4" fill="white" />
            </>
          )}
          {c.expression === 'calm' && (
            <g fill={skin}>
              <path d="M39 44h19v9H39z" />
              <path d="M62 44h19v9H62z" />
            </g>
          )}
        </>
      )}
      <path d="M59 56c-2 5-7 7-5 11 2 4 9 3 12 1" fill={skin} />
      <path
        d="M56 67q4 3 9 0"
        stroke={ink}
        strokeWidth="2.5"
        strokeLinecap="round"
        fill="none"
        opacity=".18"
      />
      {c.expression === 'laugh' ? (
        <>
          <path d="M48 74q12 19 24 0z" fill={ink} />
          <path d="M50 75h20l-3 5H53z" fill="#fffdf5" />
          <path d="M55 85q5-5 10 0" fill="#e97589" />
        </>
      ) : (
        <path
          d={c.expression === 'calm' ? 'M54 77q6 2 12 0' : 'M51 75q9 10 18 0'}
          fill="none"
          stroke={ink}
          strokeWidth="3"
          strokeLinecap="round"
        />
      )}
      {c.hair === 'crop' && (
        <path
          d="M30 42V31c0-14 14-17 29-17 18 0 31 6 31 22v7l-9-10c-11 7-24 6-35 0l-6 11z"
          fill={hair}
        />
      )}
      {c.hair === 'swoop' && (
        <path
          d="M29 46c-6-16 0-29 16-31 14-2 30 5 41-1 14 10 7 27 0 34l-9-13c-16 8-26 4-30-2z"
          fill={hair}
        />
      )}
      {c.hair === 'bob' && (
        <path
          d="M26 55V32c2-13 15-20 34-20s32 7 34 20v23l-11-7-6-18c-10 10-18 14-35 13l-7 16z"
          fill={hair}
        />
      )}
      {c.hair === 'curls' && (
        <g fill={hair}>
          <circle cx="34" cy="38" r="11" />
          <circle cx="47" cy="30" r="12" />
          <circle cx="64" cy="29" r="13" />
          <circle cx="81" cy="36" r="12" />
        </g>
      )}
      {c.hair === 'bun' && (
        <path
          d="M29 49V35c0-17 13-23 31-23s31 6 31 23v14l-9-8-5-12c-9 12-21 14-36 8z"
          fill={hair}
        />
      )}
      {c.hair === 'scarf' && (
        <>
          <path
            d="M23 60V35c0-17 15-23 37-23s37 6 37 23v27L84 41c-9-1-19-6-24-13-8 7-17 13-25 14z"
            fill={hair}
          />
          <path d="M25 72q5 19 22 23l-14 16-13-4zM95 72q-5 19-22 23l14 16 13-4z" fill={hair} />
        </>
      )}
      {c.glasses !== 'none' && (
        <g stroke={ink} strokeWidth="3.5" fill="none">
          <rect x="36" y="44" width="24" height="24" rx={c.glasses === 'round' ? 12 : 6} />
          <rect x="62" y="44" width="24" height="24" rx={c.glasses === 'round' ? 12 : 6} />
          <path d="M60 53h2M31 51h5M86 51h4" />
        </g>
      )}
      <path d="M24 117h72" stroke="#fff" strokeWidth="3" strokeLinecap="round" opacity=".12" />
    </svg>
  );
}

export function Avatar({
  name,
  color = 'violet',
  config,
  small = false,
}: {
  name: string;
  color?: string;
  config?: AvatarConfig;
  small?: boolean;
}) {
  return (
    <span
      className={`avatar avatar-character ${color} ${small ? 'small' : ''}`}
      role="img"
      aria-label={`آواتار ${name}`}
    >
      <AvatarArt config={config?.version === 1 ? config : avatarFor(name)} />
    </span>
  );
}

export function AvatarBuilder({
  value,
  background,
  onChange,
  onBackground,
}: {
  value: AvatarConfig;
  background: string;
  onChange: (value: AvatarConfig) => void;
  onBackground: (value: string) => void;
}) {
  const [part, setPart] = useState<Part>('hair');
  useEffect(() => {
    document.getElementById('avatar-tab-hair')?.focus({ preventScroll: true });
  }, []);
  const selectedPart = parts.find((p) => p.id === part)!;
  const selected = part === 'background' ? background : value[part];
  const randomize = () => {
    const next = { ...value };
    for (const p of parts) {
      const choice = p.choices[Math.floor(Math.random() * p.choices.length)].value;
      if (p.id === 'background') onBackground(choice);
      else next[p.id] = choice;
    }
    onChange(next);
  };
  return (
    <div className="avatar-builder">
      <div className="avatar-builder-preview">
        <div className="avatar-preview-halo">
          <Avatar name="شخصیت شما" config={value} color={background} />
          <span className="avatar-preview-spark">✦</span>
        </div>
        <div className="avatar-preview-caption">
          <strong>این خودِ خودتی!</strong>
          <p>یک شخصیت برای میزِ تیم شما</p>
          <button type="button" className="secondary avatar-shuffle" onClick={randomize}>
            <Shuffle size={16} />
            یک ترکیب تازه
          </button>
        </div>
      </div>
      <div className="avatar-builder-controls">
        <div className="avatar-part-tabs" role="tablist" aria-label="بخش‌های آواتار">
          {parts.map((p) => (
            <button
              type="button"
              role="tab"
              id={`avatar-tab-${p.id}`}
              aria-controls="avatar-options"
              aria-selected={part === p.id}
              tabIndex={part === p.id ? 0 : -1}
              key={p.id}
              className={part === p.id ? 'selected' : ''}
              onClick={() => setPart(p.id)}
              onKeyDown={(e) => {
                if (['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(e.key)) {
                  e.preventDefault();
                  const index = parts.findIndex((p) => p.id === part);
                  const next =
                    e.key === 'Home'
                      ? 0
                      : e.key === 'End'
                        ? parts.length - 1
                        : (index + (e.key === 'ArrowLeft' ? 1 : -1) + parts.length) % parts.length;
                  setPart(parts[next].id);
                  document.getElementById(`avatar-tab-${parts[next].id}`)?.focus();
                }
              }}
            >
              {p.label}
            </button>
          ))}
        </div>
        <div
          id="avatar-options"
          role="tabpanel"
          aria-labelledby={`avatar-tab-${part}`}
          className="avatar-options-panel"
        >
          <div className="avatar-options-heading">
            <h3>{selectedPart.label}</h3>
            <span>انتخاب کن، همین حالا ببین</span>
          </div>
          <div className={`avatar-options ${selectedPart.choices[0].color ? 'swatches' : ''}`}>
            {selectedPart.choices.map((choice) => {
              const isSelected = selected === choice.value;
              return (
                <button
                  type="button"
                  key={choice.value}
                  aria-label={`${selectedPart.label}: ${choice.label}`}
                  aria-pressed={isSelected}
                  className={`avatar-choice ${isSelected ? 'selected' : ''}`}
                  onClick={() =>
                    part === 'background'
                      ? onBackground(choice.value)
                      : onChange({ ...value, [part]: choice.value })
                  }
                >
                  {choice.color ? (
                    <span
                      className="avatar-swatch"
                      style={{ '--swatch': choice.color } as CSSProperties}
                    />
                  ) : (
                    <span className={`avatar-choice-art ${background}`}>
                      <AvatarArt config={{ ...value, [part]: choice.value }} />
                    </span>
                  )}
                  <span className="avatar-choice-label">{choice.label}</span>
                  {isSelected && <Check className="avatar-choice-check" size={14} />}
                </button>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}
