package settings

// Every setting this platform has, in one file.
//
// New keys go here rather than beside the code that reads them, because the
// question an operator asks — "what can I change?" — should have one answer,
// and because the review that matters for a setting is "should this be
// editable at all", which is easier to hold when the whole list is on one
// screen.
//
// The first four are the migration the plan asks for: values that were env-only
// and are safely dynamic. The env variable stays the fallback in every case, so
// a deployment that has never opened the console is unaffected.

const (
	// AccessMode decides whether strangers may become users of this platform.
	//
	// The one that changes the most about how the platform behaves, and the
	// reason it is the first setting rather than an afterthought: a platform
	// that provisions an account for anybody who can authenticate somewhere
	// else is a very different thing from one where somebody has to be
	// invited, and until now that difference was decided by which environment
	// variables happened to be set.
	AccessMode = "platform.access_mode"

	// SessionIdleTimeout is how long a session may go unused.
	SessionIdleTimeout = "session.idle_timeout"

	// CatalogSyncInterval is how often the app catalogue is refetched.
	CatalogSyncInterval = "catalog.sync_interval"

	// AIModel is the Gemini model the copilot asks.
	AIModel = "ai.model"

	// Maintenance closes the platform to writing.
	Maintenance = "platform.maintenance"

	// MaintenanceMessage is what people are told while it is closed.
	MaintenanceMessage = "platform.maintenance_message"
)

// The access modes.
const (
	// AccessPublic: anybody who can prove an identity gets an account.
	AccessPublic = "public"
	// AccessPrivate: only people somebody has already registered.
	AccessPrivate = "private"
)

func init() {
	Register(Spec{
		Key:     AccessMode,
		Kind:    KindEnum,
		Options: []string{AccessPrivate, AccessPublic},
		// Private is the safe default and therefore the default.
		//
		// It is also a behaviour change for a deployment that relied on
		// just-in-time provisioning through eID or a federated provider, and
		// that is the right way round: a platform that quietly starts creating
		// accounts for strangers because nobody set a variable is the failure
		// this setting exists to prevent. The console says which mode it is
		// in, and switching to public is one field and a reason.
		Default: AccessPrivate,
		// The env fallback every other setting here has, and the one this
		// setting needed most: a distribution that is public by design —
		// sso.gerege.mn — otherwise has to be switched by hand in the console
		// after every fresh deployment, and until somebody does, eID sign-in
		// answers 403 to everybody. The console still wins over it.
		Env: "PLATFORM_ACCESS_MODE",
		Description: "Хаалттай (private) горимд зөвхөн урьдчилан бүртгэгдсэн хүн нэвтэрнэ: " +
			"eID, ДАН, SSO-гоор баталгаажсан ч бүртгэлгүй хүнд шинэ данс үүсэхгүй. " +
			"Нээлттэй (public) горимд анх удаа нэвтэрсэн хүнд данс автоматаар үүснэ.",
	})

	Register(Spec{
		Key:         SessionIdleTimeout,
		Kind:        KindDuration,
		Default:     "90m",
		Env:         "SESSION_IDLE_TIMEOUT",
		Description: "Session хэдэн хугацаанд ашиглагдаагүй бол дуусах вэ. 0 нь хугацаагүй.",
	})

	Register(Spec{
		Key:         CatalogSyncInterval,
		Kind:        KindDuration,
		Default:     "1h",
		Env:         "CATALOG_SYNC_INTERVAL",
		Description: "Апп сторын каталогийг хэдэн хугацаа тутам registry-ээс шинэчлэх вэ.",
	})

	Register(Spec{
		Key:     AIModel,
		Kind:    KindString,
		Default: "",
		Env:     "GEMINI_MODEL",
		Description: "Copilot ямар Gemini загвар асуух вэ. Хоосон бол клиентийн " +
			"өөрийн анхдагч (тухайн хувилбарын flash загвар).",
	})

	Register(Spec{
		Key:     Maintenance,
		Kind:    KindBool,
		Default: "false",
		Description: "Платформ даяар зөвхөн унших горим. Хэрэглэгчид нэвтэрч, харж чадах " +
			"ч юу ч өөрчилж чадахгүй. Операторын консол үүнд хамаарахгүй.",
	})

	Register(Spec{
		Key:         MaintenanceMessage,
		Kind:        KindString,
		Default:     "",
		Description: "Засварын горимд хэрэглэгчдэд харагдах мессеж. Хоосон бол ерөнхий текст.",
	})
}
