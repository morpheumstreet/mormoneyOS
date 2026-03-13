package tools

// mutatingTools are tool names that perform side effects and count as "real work"
// for idle-turn detection. TS MUTATING_TOOLS-aligned.
var mutatingTools = map[string]bool{
	"shell": true, "exec": true, "write_file": true, "edit_own_file": true,
	"transfer_credits": true, "fund_child": true, "spawn_child": true, "start_child": true,
	"delete_sandbox": true, "create_sandbox": true,
	"install_npm_package": true, "install_skill": true, "create_skill": true, "remove_skill": true,
	"pull_upstream": true, "git_commit": true, "git_push": true, "git_branch": true, "git_clone": true,
	"send_message": true, "message_child": true,
	"update_genesis_prompt": true, "modify_heartbeat": true,
	"expose_port": true, "remove_port": true,
	"distress_signal": true, "prune_dead_children": true, "sleep": true,
	"update_soul": true, "remember_fact": true, "set_goal": true, "complete_goal": true,
	"save_procedure": true, "note_about_agent": true, "forget": true,
	"enter_low_compute": true, "switch_model": true, "review_upstream_changes": true,
}

// IsMutatingTool returns true if the tool performs side effects that count as "real work".
// Used for idle-turn detection (TS step 13 aligned).
func IsMutatingTool(name string) bool {
	return mutatingTools[name]
}
