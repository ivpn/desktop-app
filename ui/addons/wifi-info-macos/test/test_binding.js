console.log(" * Tests start");

const addon = require("../lib/binding.js");
const assert = require("assert");

function testBasic()
{    
    assert.ok(addon.LocationServicesAuthorizationStatus, 'LocationServicesAuthorizationStatus is not defined');
    assert.ok(addon.LocationServicesEnabled, 'LocationServicesEnabled is not defined');
    assert.ok(addon.LocationServicesRequestPermission, 'LocationServicesRequestPermission is not defined');
    assert.ok(addon.LocationServicesSetAuthorizationChangeCallback, 'LocationServicesSetAuthorizationChangeCallback is not defined');
    assert.ok(addon.AgentInstall, 'AgentInstall is not defined');
    assert.ok(addon.AgentUninstall, 'AgentUninstall is not defined');
    assert.ok(addon.AgentGetStatus, 'AgentGetStatus is not defined');

    //addon.LocationServicesSetAuthorizationChangeCallback(() => { console.log("callback called") });

    let agentStatus = addon.AgentGetStatus();
    console.log("AgentGetStatus: " + agentStatus + "(0:NotRegistered, 1:Enabled, 2:RequiresApproval, 3:NotFound)");
    assert.strictEqual(agentStatus, 3, `Unexpected value returned '${agentStatus}' instead of 3`);
}

assert.doesNotThrow(testBasic, undefined, "testBasic threw an expection");

console.log("Tests passed- everything looks OK!");