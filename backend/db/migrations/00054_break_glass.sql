-- Break-glass бүртгэл (CONTROL_PLANE_PLAN.md §2.4).
--
-- Нэг онцгой операторын бүртгэл: урт санамсаргүй нууц үг нь офлайн сейфэнд,
-- ердийн үед хэзээ ч хэрэглэгддэггүй. Түүнийг хэрэглэсэн агшинд бүх оператор
-- мэдэх ёстой.
--
-- Яагаад ийм зүйл хэрэгтэй вэ: платформын хамгийн эрхтэй интерфэйс нь өөрийн
-- өгөгдлийн санд түшиглэдэг. Бүх superadmin-ий TOTP төхөөрөмж алдагдсан,
-- эсвэл identity provider унасан өдөр консол руу орох зам үлдэх ёстой —
-- гэхдээ тэр зам нь чимээгүй байх ёсгүй.
--
-- Энэ багана нь ганцхан зүйл хийнэ: тэмдэглэнэ. Ямар ч эрх нэмэгддэггүй,
-- нэвтрэлт нь бусадтай яг ижил (нууц үг + TOTP). Ялгаа нь юу болохд байна:
--
--   * нэвтрэлт амжилттай болмогц ERROR түвшний лог бичигдэнэ (Loki-д очно),
--   * `cp_login_attempts_total{result="break_glass"}` өснө,
--   * тэр хэмжүүр дээр `severity: page` alert байна
--     (deploy/monitoring/prometheus/rules/control-plane.yml),
--   * `operator_audit`-д тусдаа үйлдлээр үлдэнэ.
--
-- Өөрөөр хэлбэл break-glass нь "нууц хаалга" биш, "дуут дохиотой хаалга".

-- +goose Up

ALTER TABLE operator_accounts
    ADD COLUMN IF NOT EXISTS break_glass BOOLEAN NOT NULL DEFAULT FALSE;

-- Ийм бүртгэл нэгээс олон байх ёсгүй. Хэсэгчилсэн unique индекс нь түүнийг
-- баталгаажуулна: хоёр дахийг үүсгэх оролдлого DB дээр унана.
--
-- Яагаад чухал вэ: break-glass-ийн бүх утга нь "ердийн үед хэрэглэгддэггүй"
-- гэдэгт байгаа. Хоёр, гурав болмогц тэдгээрийн аль нэг нь хэн нэгний ердийн
-- бүртгэл болж хувирдаг — тэр агшинд дохио нь худал дуугарч эхэлж, хүмүүс
-- үл тоомсорлож сурна.
CREATE UNIQUE INDEX IF NOT EXISTS idx_operator_accounts_one_break_glass
    ON operator_accounts ((break_glass)) WHERE break_glass;

-- +goose Down

DROP INDEX IF EXISTS idx_operator_accounts_one_break_glass;
ALTER TABLE operator_accounts DROP COLUMN IF EXISTS break_glass;
