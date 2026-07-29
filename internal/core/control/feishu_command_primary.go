package control

var feishuPrimaryCommandSpec = feishuCommandSpec{
	definition: FeishuCommandDefinition{
		ID:               FeishuCommandPrimary,
		GroupID:          FeishuCommandGroupCommonTools,
		Title:            "群主机器人",
		CanonicalSlash:   "/primary",
		CanonicalMenuKey: "primary",
		ArgumentKind:     FeishuCommandArgumentChoice,
		ArgumentFormHint: "status",
		ArgumentFormNote: "输入 on / off / status / refresh。",
		ArgumentSubmit:   "执行",
		Description:      "设置或查看本群承接未 @ 普通消息的主机器人。",
		Examples:         []string{"/primary on", "/primary off", "/primary status", "/primary refresh"},
		Options: []FeishuCommandOption{
			commandOption("/primary", "primary", "on", "设为本群主机器人", "把当前机器人设为本群主机器人。"),
			commandOption("/primary", "primary", "off", "取消主机器人", "取消当前机器人作为本群主机器人。"),
			commandOption("/primary", "primary", "status", "查看状态", "查看本群主机器人和权限状态。"),
			commandOption("/primary", "primary", "refresh", "刷新权限", "刷新当前机器人的群普通消息权限状态。"),
		},
		ShowInHelp: true,
		ShowInMenu: true,
	},
	textPrefixes: []feishuCommandPrefixMatch{
		{alias: "/primary", kind: ActionPrimaryCommand},
	},
	menuExact: []feishuCommandMatch{
		{alias: "primary", action: Action{Kind: ActionPrimaryCommand, Text: "/primary status"}},
		{alias: "primaryon", action: Action{Kind: ActionPrimaryCommand, Text: "/primary on"}},
		{alias: "primaryoff", action: Action{Kind: ActionPrimaryCommand, Text: "/primary off"}},
		{alias: "primarystatus", action: Action{Kind: ActionPrimaryCommand, Text: "/primary status"}},
		{alias: "primaryrefresh", action: Action{Kind: ActionPrimaryCommand, Text: "/primary refresh"}},
	},
	menuDynamic: []feishuCommandDynamicMenuMatch{
		{prefix: "primary_", kind: ActionPrimaryCommand, parseArgument: normalizePrimaryMenuArgument},
		{prefix: "primary-", kind: ActionPrimaryCommand, parseArgument: normalizePrimaryMenuArgument},
	},
}
