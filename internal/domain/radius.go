package domain

import (
	"time"
)

// RADIUS related models

// RadiusProfile RADIUS billing profile
type RadiusProfile struct {
	ID             int64  `json:"id,string" form:"id"` // Primary key ID
	TenantID       int64  `json:"-" form:"-" gorm:"not null;default:1;index"`
	NodeId         int64  `json:"node_id,string" form:"node_id"`            // Node ID
	Name           string `json:"name" form:"name"`                         // Profile name
	Status         string `gorm:"index" json:"status" form:"status"`        // Profile status: 0=disabled 1=enabled
	AddrPool       string `json:"addr_pool" form:"addr_pool"`               // Address pool
	ActiveNum      int    `json:"active_num" form:"active_num"`             // Concurrent sessions
	UpRate         int    `json:"up_rate" form:"up_rate"`                   // Upload rate in Kb
	DownRate       int    `json:"down_rate" form:"down_rate"`               // Download rate in Kb
	Domain         string `json:"domain" form:"domain"`                     // Domain, corresponds to NAS device domain attribute, e.g., Huawei domain_code
	IPv6PrefixPool string `json:"ipv6_prefix_pool" form:"ipv6_prefix_pool"` // IPv6 prefix pool name for NAS-side allocation
	// DelegatedIpv6PrefixPool names the DHCPv6 Prefix-Delegation pool the NAS uses
	// to allocate a Delegated-IPv6-Prefix (RFC 6911 Delegated-IPv6-Prefix-Pool,
	// attribute 171). Per RFC 6911 §2.4 it is distinct from IPv6PrefixPool
	// (Framed-IPv6-Pool, used for SLAAC) and must not reuse the same value.
	DelegatedIpv6PrefixPool string `json:"delegated_ipv6_prefix_pool" form:"delegated_ipv6_prefix_pool"`
	// RadiusClass is the RFC 2865 Class (Type 25) value copied into Access-Accept.
	// The string is emitted as-is (max 253 octets). ocserv groupconfig expects
	// "OU=group1;group2". Empty or "NA" means the attribute is omitted.
	RadiusClass string    `json:"radius_class" form:"radius_class"`
	BindMac     int       `json:"bind_mac" form:"bind_mac"`   // Bind MAC
	BindVlan    int       `json:"bind_vlan" form:"bind_vlan"` // Bind VLAN
	Remark      string    `json:"remark" form:"remark"`       // Remark
	CreatedAt   time.Time `json:"created_at" form:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" form:"updated_at"`
}

// TableName Specify table name
func (RadiusProfile) TableName() string {
	return "radius_profile"
}

// RadiusUser RADIUS Authentication account
type RadiusUser struct {
	ID             int64  `json:"id,string" form:"id"`                                                                    // Primary key ID
	NodeId         int64  `json:"node_id,string" form:"node_id"`                                                          // Node ID
	ProfileId      int64  `gorm:"index" json:"profile_id,string" form:"profile_id"`                                       // RADIUS profile ID
	Realname       string `json:"realname" form:"realname"`                                                               // Contact name
	Email          string `json:"email" form:"email"`                                                                     // Email address
	Mobile         string `json:"mobile" form:"mobile"`                                                                   // Contact phone
	Address        string `json:"address" form:"address"`                                                                 // Contact address
	Username       string `json:"username" gorm:"uniqueIndex:udx_radius_user_tenant_username,priority:2" form:"username"` // Account name
	TenantID       int64  `json:"-" form:"-" gorm:"not null;default:1;uniqueIndex:udx_radius_user_tenant_username,priority:1;index"`
	Password       string `json:"password" form:"password"`                  // Password
	AddrPool       string `json:"addr_pool" form:"addr_pool"`                // Address pool
	ActiveNum      int    `gorm:"index" json:"active_num" form:"active_num"` // Concurrent sessions
	UpRate         int    `json:"up_rate" form:"up_rate"`                    // Upload rate
	DownRate       int    `json:"down_rate" form:"down_rate"`                // Download rate
	Vlanid1        int    `json:"vlanid1" form:"vlanid1"`                    // VLAN ID 1
	Vlanid2        int    `json:"vlanid2" form:"vlanid2"`                    // VLAN ID 2
	IpAddr         string `json:"ip_addr" form:"ip_addr"`                    // Static IP
	IpV6Addr       string `json:"ipv6_addr" form:"ipv6_addr"`                // Static IPv6 address
	MacAddr        string `json:"mac_addr" form:"mac_addr"`                  // MAC address
	Domain         string `json:"domain" form:"domain"`                      // Domain name for vendor-specific features (e.g., Huawei domain)
	IPv6PrefixPool string `json:"ipv6_prefix_pool" form:"ipv6_prefix_pool"`  // IPv6 prefix pool name (inherited from profile or user-specific)
	// DelegatedIpv6Prefix is a static IPv6 prefix delegated to the subscriber via
	// DHCPv6-PD and issued as the RADIUS Delegated-IPv6-Prefix attribute (RFC 4818,
	// attribute 123), e.g. "2001:db8:1234::/48". It is per-user because a delegated
	// prefix routes to a single subscriber; an empty value means "not assigned".
	DelegatedIpv6Prefix string `json:"delegated_ipv6_prefix" form:"delegated_ipv6_prefix"`
	// DelegatedIpv6PrefixPool names the DHCPv6 Prefix-Delegation pool the NAS uses
	// to allocate a Delegated-IPv6-Prefix (RFC 6911 Delegated-IPv6-Prefix-Pool,
	// attribute 171). An empty value inherits from the linked RadiusProfile in
	// dynamic link mode; per RFC 6911 §2.4 it is distinct from IPv6PrefixPool.
	DelegatedIpv6PrefixPool string `json:"delegated_ipv6_prefix_pool" form:"delegated_ipv6_prefix_pool"`
	// RadiusClass is the RFC 2865 Class (Type 25) value written into Access-Accept.
	// A non-empty user value overrides the linked profile; empty/"NA" inherits in
	// dynamic link mode. The string is emitted as-is (max 253 octets).
	RadiusClass     string    `json:"radius_class" form:"radius_class"`
	BindVlan        int       `json:"bind_vlan" form:"bind_vlan"`                 // Bind VLAN
	BindMac         int       `json:"bind_mac" form:"bind_mac"`                   // Bind MAC
	ProfileLinkMode int       `json:"profile_link_mode" form:"profile_link_mode"` // 0=static (snapshot), 1=dynamic (real-time from profile)
	ExpireTime      time.Time `gorm:"index" json:"expire_time"`                   // Expiration time
	Status          string    `gorm:"index" json:"status" form:"status"`          // Status: enabled | disabled
	Remark          string    `json:"remark" form:"remark"`                       // Remark
	OnlineCount     int       `json:"online_count" gorm:"-:migration;<-:false"`
	LastOnline      time.Time `json:"last_online"`
	CreatedAt       time.Time `gorm:"index" json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TableName Specify table name
func (RadiusUser) TableName() string {
	return "radius_user"
}

// RadiusOnline stores active online session state tracked from accounting packets.
type RadiusOnline struct {
	ID                  int64     `json:"id,string"` // Primary key ID
	TenantID            int64     `json:"-" gorm:"not null;default:1;uniqueIndex:udx_radius_online_tenant_session,priority:1;index"`
	Username            string    `gorm:"index" json:"username"`
	NasId               string    `json:"nas_id"`
	NasAddr             string    `json:"nas_addr"`
	NasPaddr            string    `json:"nas_paddr"`
	SessionTimeout      int       `json:"session_timeout"`
	FramedIpaddr        string    `json:"framed_ipaddr"`
	FramedNetmask       string    `json:"framed_netmask"`
	FramedIpv6Prefix    string    `json:"framed_ipv6_prefix"`
	FramedIpv6Address   string    `json:"framed_ipv6_address"`
	DelegatedIpv6Prefix string    `json:"delegated_ipv6_prefix"`
	MacAddr             string    `json:"mac_addr"`
	NasPort             int64     `json:"nas_port,string"`
	NasClass            string    `json:"nas_class"`
	NasPortId           string    `json:"nas_port_id"`
	NasPortType         int       `json:"nas_port_type"`
	ServiceType         int       `json:"service_type"`
	AcctSessionId       string    `gorm:"uniqueIndex:udx_radius_online_tenant_session,priority:2" json:"acct_session_id"`
	AcctSessionTime     int       `json:"acct_session_time"`
	AcctInputTotal      int64     `json:"acct_input_total,string"`
	AcctOutputTotal     int64     `json:"acct_output_total,string"`
	AcctInputPackets    int       `json:"acct_input_packets"`
	AcctOutputPackets   int       `json:"acct_output_packets"`
	AcctStartTime       time.Time `gorm:"index" json:"acct_start_time"`
	LastUpdate          time.Time `json:"last_update"`
}

// TableName Specify table name
func (RadiusOnline) TableName() string {
	return "radius_online"
}

// RadiusSessionActionAudit records a durable audit trail for operator-triggered
// dynamic-authorization actions (Disconnect / CoA) on online sessions.
type RadiusSessionActionAudit struct {
	ID             int64     `json:"id,string"`
	TenantID       int64     `json:"-" gorm:"not null;default:1;index"`
	SessionID      int64     `gorm:"index" json:"session_id,string"`
	AcctSessionID  string    `gorm:"index" json:"acct_session_id"`
	Action         string    `gorm:"index" json:"action"`
	Username       string    `gorm:"index" json:"username"`
	NasAddr        string    `gorm:"index" json:"nas_addr"`
	Target         string    `json:"target"`
	OperatorID     int64     `gorm:"index" json:"operator_id,string"`
	OperatorName   string    `gorm:"index" json:"operator_name"`
	OperatorIP     string    `json:"operator_ip"`
	Success        bool      `gorm:"index" json:"success"`
	ResponseCode   string    `json:"response_code"`
	ErrorCause     int       `json:"error_cause"`
	ErrorCauseText string    `json:"error_cause_text"`
	Attempts       int       `json:"attempts"`
	RTTMillis      int64     `json:"rtt_ms"`
	TimedOut       bool      `gorm:"index" json:"timed_out"`
	Error          string    `json:"error"`
	TriggeredAt    time.Time `gorm:"index" json:"triggered_at"`
	CreatedAt      time.Time `json:"created_at"`
}

// TableName specifies the CoA session action audit table name.
func (RadiusSessionActionAudit) TableName() string {
	return "radius_session_action_audit"
}

// RadiusSessionAction is a durable, idempotent unit of required NAS work whose
// desired state may not have been reached yet, for example a Disconnect-Request
// after a billing suspension when the NAS is offline. The scheduler claims a
// row, attempts the action, and either records success or schedules a bounded
// retry. An operator can retry a failed row manually.
//
// IdempotencyKey is unique per tenant so repeated billing runs cannot enqueue
// duplicate Disconnect-Requests for the same online session. ClaimToken is an
// internal fencing value that prevents a stale or concurrent worker from
// finalizing an action claimed by another run; it is never exposed by the API.
type RadiusSessionAction struct {
	ID                 int64      `json:"id,string"`
	TenantID           int64      `json:"-" gorm:"not null;default:1;uniqueIndex:udx_radius_session_action_key,priority:1;index"`
	IdempotencyKey     string     `json:"-" gorm:"size:128;not null;uniqueIndex:udx_radius_session_action_key,priority:2"`
	SessionID          int64      `gorm:"index" json:"session_id,string"`
	AcctSessionID      string     `gorm:"index;size:64" json:"acct_session_id"`
	Username           string     `gorm:"index;size:128" json:"username"`
	NasAddr            string     `gorm:"index;size:64" json:"nas_addr"`
	Action             string     `gorm:"index;size:16" json:"action"`
	Reason             string     `gorm:"index;size:32" json:"reason"`
	Status             string     `gorm:"index;size:16;not null;default:pending" json:"status"`
	Attempts           int        `gorm:"not null;default:0" json:"attempts"`
	MaxAttempts        int        `gorm:"not null;default:8" json:"max_attempts"`
	ManualRetries      int        `gorm:"not null;default:0" json:"manual_retries"`
	NextRetryAt        time.Time  `gorm:"index" json:"next_retry_at"`
	LastAttemptAt      *time.Time `json:"last_attempt_at"`
	LastResponseCode   string     `gorm:"size:32" json:"last_response_code"`
	LastErrorCause     int        `json:"last_error_cause"`
	LastErrorCauseText string     `gorm:"size:128" json:"last_error_cause_text"`
	LastError          string     `gorm:"size:512" json:"last_error"`
	LastTimedOut       bool       `json:"last_timed_out"`
	LastWireAttempts   int        `json:"last_wire_attempts"`
	LastRTTMillis      int64      `json:"last_rtt_ms"`
	SucceededAt        *time.Time `json:"succeeded_at"`
	ClaimToken         string     `gorm:"size:64" json:"-"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// TableName specifies the durable session action table name.
func (RadiusSessionAction) TableName() string {
	return "radius_session_action"
}

// RadiusAccounting stores finalized accounting records for session history.
type RadiusAccounting struct {
	ID                  int64     `json:"id,string"` // Primary key ID
	TenantID            int64     `json:"-" gorm:"not null;default:1;index"`
	Username            string    `gorm:"index" json:"username"`
	AcctSessionId       string    `gorm:"index" json:"acct_session_id"`
	NasId               string    `json:"nas_id"`
	NasAddr             string    `json:"nas_addr"`
	NasPaddr            string    `json:"nas_paddr"`
	SessionTimeout      int       `json:"session_timeout"`
	FramedIpaddr        string    `json:"framed_ipaddr"`
	FramedNetmask       string    `json:"framed_netmask"`
	FramedIpv6Prefix    string    `json:"framed_ipv6_prefix"`
	FramedIpv6Address   string    `json:"framed_ipv6_address"`
	DelegatedIpv6Prefix string    `json:"delegated_ipv6_prefix"`
	MacAddr             string    `json:"mac_addr"`
	NasPort             int64     `json:"nas_port,string"`
	NasClass            string    `json:"nas_class"`
	NasPortId           string    `json:"nas_port_id"`
	NasPortType         int       `json:"nas_port_type"`
	ServiceType         int       `json:"service_type"`
	AcctSessionTime     int       `json:"acct_session_time"`
	AcctInputTotal      int64     `json:"acct_input_total,string"`
	AcctOutputTotal     int64     `json:"acct_output_total,string"`
	AcctInputPackets    int       `json:"acct_input_packets"`
	AcctOutputPackets   int       `json:"acct_output_packets"`
	LastUpdate          time.Time `json:"last_update"`
	AcctStartTime       time.Time `gorm:"index" json:"acct_start_time"`
	AcctStopTime        time.Time `gorm:"index" json:"acct_stop_time"`
}

// TableName Specify table name
func (RadiusAccounting) TableName() string {
	return "radius_accounting"
}
