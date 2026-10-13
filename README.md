# هم‌تخمین — Planning Poker

پلتفرم Go + React برای تخمین گروهی، با UX الهام‌گرفته از [Kollabe](https://kollabe.com/planning-poker).

- ورود/ثبت‌نام با شماره موبایل ایرانی و OTP، پروفایل و آواتار
- سازندهٔ آواتار کارتونی با انتخاب چهره، مو، پوست، عینک، لباس و پس‌زمینه؛ ذخیره در پروفایل و نمایش زنده در اتاق
- اتاق عمومی/خصوصی؛ نمایش اتاق‌های عمومی در صفحهٔ اصلی، پیوستن مستقیم و دعوت با لینک برای اتاق خصوصی
- ذخیره‌سازی PostgreSQL با queryهای sqlc، migrationهای Goose، فرمان‌های Cobra و تنظیمات YAML
- رأی‌گیری زنده، حضور اعضا، کارت‌های فیبوناچی و اندازه تی‌شرت
- پنهان بودن رأی دیگران تا نمایش توسط گرداننده، شروع مجدد و ثبت تخمین نهایی
- تسک‌های متوالی، تاریخچه جلسه و خروجی CSV
- فارسی، RTL، فونت محلی وزیر (Vazir)، چیدمان موبایل، تم روشن و تیره
- شش تم رنگی اتاق: بنفش، صورتی، آبی، سبز نعنایی، کهربایی و مرجانی؛ انتخاب هنگام ساخت و تغییر زنده توسط سازنده

## اجرای سریع محلی

نیازمندی: Go 1.23+، Node 22.12+ و PostgreSQL یا Docker.

```bash
make setup
make db
make migrate
make run
```

آدرس: http://127.0.0.1:8080 — اتاق نمونه: http://127.0.0.1:8080/demo

در حالت development کد OTP در پنجره ورود نمایش داده می‌شود و پیامک واقعی ارسال نمی‌شود. شماره تست: `09120000123`. کدها تصادفی هستند؛ کد ثابت یا ورود bypass وجود ندارد. اتاق نمونه اعضای شبیه‌سازی‌شده دارد؛ اتاق‌های واقعی همیشه از API Go استفاده می‌کنند.

داده‌ها در جدول‌های PostgreSQL ذخیره می‌شوند و با restart باقی می‌مانند. `make db` در صورت نصب ابزارهای PostgreSQL، کلاستر اختصاصی را در `backend/data/postgres` روی `127.0.0.1:55432` بالا می‌آورد؛ در غیر این صورت از سرویس PostgreSQL در Compose استفاده می‌کند. تنظیمات برنامه در `backend/config/config.yml` است. بعد از آماده‌سازی دیتابیس، `make run` برنامه را با فرمان Cobra `serve` اجرا می‌کند. دستورهای کامل و انتقال فایل قبلی در [راهنمای PostgreSQL و CLI](docs/postgres-and-rooms.md) آمده‌اند.

برای توسعه با hot reload، پس از نصب وابستگی‌ها:

```bash
make dev
```

React روی `http://127.0.0.1:5173` اجرا می‌شود و `/api` را به Go پروکسی می‌کند.

## دسترسی از دستگاه‌های شبکه محلی

سرور روی `0.0.0.0:8080` گوش می‌دهد. از دستگاه دیگری در همین شبکه، آدرس زیر را باز کنید:

```text
http://192.168.15.87:8080
```

برای `make dev` از `http://192.168.15.87:5173` استفاده کنید. این دو origin همراه با localhost در `backend/config/config.yml`، Docker و `.env.example` مجازند. `0.0.0.0` آدرس bind است؛ در مرورگر و لینک دعوت از IP واقعی کامپیوتر استفاده کنید. `192.168.14.1` در این شبکه gateway است.

اگر IP کامپیوتر تغییر کرد، `server.allowed_origins` در YAML یا `ALLOWED_ORIGINS` را با IP جدید تنظیم و برنامه را restart کنید. اجرای مستقیم Go متغیرهای environment را می‌خواند؛ برای override پیش‌فرض می‌توانید قبل از `make run` آن‌ها را export کنید. Docker مقدار `.env` را می‌خواند.

## Docker و PostgreSQL

```bash
cp .env.example .env
# POSTGRES_PASSWORD را در .env تنظیم کنید.
docker compose up --build
```

React ساخته شده را همان سرور Go ارائه می‌کند. سرویس `migrate` قبل از API، migrationهای Goose را اجرا می‌کند. PostgreSQL volume مستقل دارد و فقط از loopback روی پورت 55432 در دسترس است. برای اجرای کامل Docker، کلاستر native را ابتدا با `./scripts/postgres.sh stop` متوقف کنید تا پورت آزاد باشد. برای بارگذاری env در اجرای مستقیم، متغیرها را export کنید؛ Go فایل `.env` را خودکار نمی‌خواند. اگر رمز PostgreSQL شامل کاراکترهای ویژه است، `DATABASE_URL` کامل با رمز URL-encoded را در `.env` وارد کنید.

## ساختار مشابه HermesX

```text
backend/
  app/                      # Fx wiring, Fiber lifecycle, safe errors
  main.go                   # Cobra entrypoint: go run . serve
  cmd/                      # serve, pg_create, pg_migrate, pg_rollback, pg_status, pg_import
  config/config.yml         # YAML server/db/sms/realtime settings
  handler/controllers/rest/ # transport-only HTTP controllers
  handler/dtos/             # request DTOs
  handler/routers/          # routing, auth, origin checks, rate limits
  internal/domain/          # users, sessions, rooms, rounds, cards
  internal/services/        # OTP, permissions, voting lifecycle
  internal/repositories/    # transactional PostgreSQL adapter; file fixtures/import
  internal/database/postgres/ # Goose migrations, SQL queries, generated sqlc code
  internal/realtime/        # room subscriptions and presence
  internal/producers/       # development / SMS webhook adapter
  internal/config/          # Viper YAML + environment overrides
  pkg/                      # cryptographic codes, tokens, hashing
frontend/src/               # React + TypeScript + Vite
```

ساختار لایه‌ها، Fiber، Fx و Zap از ساختار فعلی `/home/parsa/Desktop/HermesX` الگوبرداری شده‌اند. سورس HermesX تغییر نکرده است. وابستگی به gRPC، NATS یا provider پیامک برای اجرای محلی لازم نیست. انتقال realtime با **SSE** برای تغییرات سرور و HTTP برای فرمان‌ها انجام می‌شود؛ سشن HttpOnly مانع قرار گرفتن token در URL یا localStorage می‌شود.

## قواعد امنیت و جلسه

OTP سه دقیقه اعتبار، ۶ رقم تصادفی، حداکثر ۵ تلاش، فاصله ارسال ۶۰ ثانیه و حداکثر ۵ ارسال در ساعت برای هر شماره دارد. فقط hash کد با salt تصادفی ذخیره می‌شود و کد پس از استفاده نامعتبر می‌شود. سشن ۷ روز اعتبار دارد؛ token خام در دیتابیس ذخیره نمی‌شود. API شماره تلفن سایر اعضا را برنمی‌گرداند. کلاینت نمی‌تواند نقش گرداننده، رأی شخص دیگر یا وضعیت round را تعیین کند.

هر اتاق حداکثر ۲۰ عضو و ۲۰۰ تسک دارد. تکمیل پروفایل برای ساخت/ورود به اتاق ضروری است. هنگام رأی‌گیری فقط وضعیت «رأی داده» و رأی خود کاربر ارسال می‌شود؛ مقدار رأی دیگران و آمار آن‌ها تا reveal از پاسخ HTTP و SSE حذف می‌شود. رأی با شناسه round ارسال می‌شود تا رأی دیررس وارد round بعدی نشود. رأی پس از reveal تغییر نمی‌کند. گرداننده می‌تواند reset کند، تخمین نهایی را ثبت کند، تسک بعدی را شروع کند یا جلسه را ببندد. نتایج جلسه بسته همچنان در دسترس اعضای قبلی است.

نوع دسترسی پیش‌فرض اتاق خصوصی است؛ اتاق‌های قدیمی هم خصوصی باقی می‌مانند. سازنده می‌تواند قبل از پایان جلسه نوع دسترسی را تغییر دهد. فهرست عمومی برای کاربر واردنشده هم قابل مشاهده است؛ پیوستن به اتاق به ورود و پروفایل نیاز دارد. فقط اطلاعات خلاصهٔ اتاق در این فهرست نمایش داده می‌شود. با پایان جلسه، اتاق از فهرست عمومی حذف می‌شود و پیام پایانی SSE به اعضا می‌رسد، اتصال‌ها پایان می‌یابند و subscriptionها آزاد می‌شوند. بعد از ۲۴ ساعت بدون فعالیت اتاق هم اتصال زنده بسته می‌شود؛ خود اتاق و داده‌ها باقی می‌مانند و دکمهٔ «اتصال دوباره» آن را فعال می‌کند. Heartbeat و خواندن تاریخچه، فعالیت محسوب نمی‌شوند.

حضور در میز و رأی‌گیری به اتصال زندهٔ مرورگر وابسته است. با قطع اتصال، صفحهٔ اتصال مجدد نمایش داده می‌شود و کارت‌ها در دسترس نیستند. فقط افراد متصل در فهرست اعضای حاضر، ظرفیت ۲۰ نفر و نتیجهٔ دور جاری حساب می‌شوند. چند تب یک کاربر یک صندلی دارند و تب پس‌زمینه حضور را حفظ می‌کند. عضویت و نتایج ثبت‌شده برای بازگشت و تاریخچه باقی می‌مانند. تشخیص قطع ارتباط برای دیگر اعضا با heartbeat سرور انجام می‌شود.

## اتصال پیامک و استقرار

برای اتصال provider، `SMS_WEBHOOK_URL` و `SMS_WEBHOOK_TOKEN` را تنظیم کنید. آداپتر به endpoint شما درخواست HTTPS می‌دهد:

```json
{"phone":"+989123456789","code":"123456","purpose":"login","message":"کد ورود هم‌تخمین: 123456"}
```

هدر: `Authorization: Bearer <SMS_WEBHOOK_TOKEN>`؛ پاسخ موفق باید 2xx باشد. یک gateway برای provider مورد نظر می‌تواند این قرارداد را به کاوه‌نگار یا سرویس دیگر تبدیل کند. بدون تنظیم provider، هیچ پیامک واقعی ارسال نمی‌شود.

در production، این مقادیر **اجباری** هستند:

```text
APP_ENV=production
DATABASE_URL=postgres://...
COOKIE_SECURE=true
ALLOWED_ORIGINS=https://your-domain.example
SMS_WEBHOOK_URL=https://your-sms-gateway.example/send
SMS_WEBHOOK_TOKEN=...
```

TLS را در reverse proxy فعال کنید، `/api` و React را روی یک origin نگه دارید، buffering و caching را برای `/api/rooms/:id/events` غیرفعال کنید و timeout اتصال SSE را بلند تنظیم کنید. نمونه Nginx در `docs/nginx.conf` است. API در Compose روی همهٔ رابط‌ها برای دسترسی LAN ارائه می‌شود؛ PostgreSQL فقط روی loopback است. برای production از دامنه/HTTPS و reverse proxy استفاده کنید.

این نسخه برای **یک replica از API** طراحی شده است. کاربران، OTPها، سشن‌ها، اتاق‌ها، اعضا، تسک‌ها و رأی‌ها جدول‌های جدا دارند؛ فقط تنظیمات آواتار JSONB است. Queryهای برنامه با sqlc تولید می‌شوند و تغییرات هر اتاق یا ورود با تراکنش و advisory lock مربوط به همان منبع انجام می‌شود. حضور و پخش realtime در حافظهٔ API است؛ برای چند replica باید event bus مشترک اضافه شود. برای انتقال دادهٔ فایل قبلی، بعد از migration و قبل از serve از `go run . pg_import --file data/state.json` استفاده کنید. فرمان import دیتابیس غیرخالی را بازنویسی نمی‌کند.

## بررسی

```bash
make test
python3 scripts/smoke.py http://127.0.0.1:8080
```

Smoke فقط روی سرور development اجرا می‌شود؛ حساب‌ها و اتاق تست به نام `QA smoke` می‌سازد و جلسه را در پایان می‌بندد. تست‌های Go مصرف یک‌باره OTP، انقضا، محدودیت تلاش، رأی‌های هم‌زمان، پنهان بودن رأی‌ها، مجوز گرداننده و چرخه round را پوشش می‌دهند. تست adapter PostgreSQL با `TEST_DATABASE_URL` فعال می‌شود و نیاز به دیتابیس اختصاصی تست دارد.

[قرارداد API](docs/api.md) · [نتیجه بررسی‌ها](docs/verification.md)

فونت اصلی وزیر v30.1.0 در `frontend/public/fonts/vazir` همراه با مجوز OFL نگهداری می‌شود؛ بارگذاری فونت به CDN وابسته نیست. [گزارش بررسی ریسپانسیو](docs/responsive-review.md) شامل اندازه‌های تست‌شده، اصلاحات و تصاویر موبایل است.

برای ساخت شخصیت: حساب کاربری ← ویرایش پروفایل ← «شخصیتت را بساز». ظاهر آواتار با SVG محلی ساخته می‌شود و تصویر از سرویس خارجی دریافت نمی‌کند. [بررسی آواتار و تم تیره](docs/avatar-review.md).
