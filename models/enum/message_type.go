package enum

type MessageType int8

const (
	MessageTypeAuditPass   MessageType = 1 // 审核通过
	MessageTypeAuditReject MessageType = 2 // 审核驳回
	MessageTypeFollow      MessageType = 3 // 被关注
)
