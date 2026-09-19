package discord

import (
	"encoding/json"
	"slices"
	"sort"
	"strings"

	"github.com/CodeMeAPixel/Mellow/internal/dashboard"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

func (b *Bot) Directory() dashboard.Directory {
	return dashboard.Directory{
		HasGuild: func(id int64) bool {
			if b.client == nil {
				return false
			}
			_, ok := b.client.Caches.Guild(snowflake.ID(id))
			return ok
		},
		Channels: func(id int64) []dashboard.Channel {
			out := []dashboard.Channel{}
			if b.client == nil {
				return out
			}
			for ch := range b.client.Caches.Channels() {
				if int64(ch.GuildID()) != id {
					continue
				}
				if ch.Type() != discord.ChannelTypeGuildText && ch.Type() != discord.ChannelTypeGuildNews {
					continue
				}
				out = append(out, dashboard.Channel{ID: ch.ID().String(), Name: ch.Name()})
			}
			sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
			return out
		},
		Roles: func(id int64) []dashboard.Role {
			out := []dashboard.Role{}
			if b.client == nil {
				return out
			}
			for r := range b.client.Caches.Roles(snowflake.ID(id)) {
				if r.Managed || r.ID == snowflake.ID(id) {
					continue
				}
				out = append(out, dashboard.Role{ID: r.ID.String(), Name: r.Name})
			}
			sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
			return out
		},
	}
}

var permNames = []struct {
	perm discord.Permissions
	name string
}{
	{discord.PermissionManageGuild, "Manage Server"},
	{discord.PermissionAdministrator, "Administrator"},
	{discord.PermissionManageChannels, "Manage Channels"},
	{discord.PermissionManageRoles, "Manage Roles"},
	{discord.PermissionModerateMembers, "Timeout Members"},
}

func permissionLabels(perms []discord.Permissions) []string {
	labels := []string{}
	var all discord.Permissions
	for _, p := range perms {
		all = all.Add(p)
	}
	for _, pn := range permNames {
		if all.Has(pn.perm) {
			labels = append(labels, pn.name)
		}
	}
	return labels
}

func (b *Bot) CommandInfos() []dashboard.CommandInfo {
	out := []dashboard.CommandInfo{}
	for _, c := range b.commands {
		if c.Private || c.OwnerOnly || c.Category == "Stub" {
			continue
		}
		info := dashboard.CommandInfo{
			Name:            c.Name,
			Description:     c.Description,
			Category:        c.Category,
			GuildOnly:       c.GuildOnly,
			CooldownSeconds: int(c.Cooldown.Seconds()),
			Permissions:     permissionLabels(c.RequiredPerms),
			RequiredRoles:   slices.Clone(c.RequiredRoles),
		}
		if c.PremiumCooldown > 0 {
			info.PlusCooldownSec = int(c.PremiumCooldown.Seconds())
		}
		if len(c.Options) > 0 {
			if raw, err := json.Marshal(c.Options); err == nil {
				info.Options = raw
			}
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}
		return strings.Compare(out[i].Name, out[j].Name) < 0
	})
	return out
}
