var fs = require('fs');
var ip = require('ip');

var Steam = require('steam'),
    steamClient = new Steam.SteamClient(),
    steamUser = new Steam.SteamUser(steamClient),
    steamGC = new Steam.SteamGameCoordinator(steamClient, 730),
    csgo = require('csgo'),
    CSGO = new csgo.CSGOClient(steamUser, steamGC, false);


steamClient.on('connected', function() {
  console.log('connected.');
  steamUser.logOn({
    account_name: process.env.STEAM_USER,
    password: process.env.STEAM_PASS,
    sha_sentryfile: fs.readFileSync('sentry.hash')
  });
});

var Long = require('bytebuffer').Long;

steamClient.on('logOnResponse', function() {
  CSGO.launch();

  CSGO.on('ready', function() {
    CSGO.requestCurrentLiveGames();
  });

  CSGO.on('matchList', function(list) {
    if (list.matches.length > 0) {
      console.log('Found some matches');
      var match = list.matches[0];
      CSGO.requestWatchInfoFriends({
        serverid: new Long(match.watchablematchinfo.server_id.low, match.watchablematchinfo.server_id.high, match.watchablematchinfo.server_id.unsigned).toString(),
        matchid: new Long(match.matchid.low, match.matchid.high, match.matchid.unsigned).toString()
      });
    }
  });

  CSGO.on('watchList', function(watch) {
    console.log(watch);
    var info = watch.watchable_match_infos[0];
    console.log('GOTV available @ ', ip.fromLong(info.server_ip) + ':' + info.tv_port);
    console.log('Password: ', info.tv_watch_password.toString('utf8'));
  });
});

steamClient.connect();