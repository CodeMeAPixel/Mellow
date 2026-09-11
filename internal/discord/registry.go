package discord

func (b *Bot) buildRegistry() []*Command {
	var cmds []*Command
	cmds = append(cmds, infoCommands()...)
	cmds = append(cmds, checkinCommands()...)
	cmds = append(cmds, copingCommand())
	cmds = append(cmds, crisisCommand())
	cmds = append(cmds, userCommands()...)
	cmds = append(cmds, funCommands()...)
	cmds = append(cmds, guildCommands()...)
	cmds = append(cmds, guildDiagCommands()...)
	cmds = append(cmds, contextCommands()...)
	cmds = append(cmds, wordgameCommands()...)
	cmds = append(cmds, memeCommands()...)
	cmds = append(cmds, adminCommands()...)
	cmds = append(cmds, supportManageCommands()...)
	cmds = append(cmds, toolsCommands()...)
	cmds = append(cmds, changelogCommands()...)
	cmds = append(cmds, upgradeCommand())
	return cmds
}
