#!/usr/bin/php
<?php

function curl_get($url) {
        $curl = curl_init();
        curl_setopt($curl, CURLOPT_URL, $url);
        curl_setopt($curl, CURLOPT_RETURNTRANSFER, true);
        $server_output = curl_exec($curl);
        curl_close($curl);
	return $server_output;
}

if (empty($argv[1])) {
	echo "Usage: ".$argv[0]." metrics_url\n";
	exit(1);
}

$url = $argv[1];

$logged = array();


while(true) {

	$res = curl_get($url);
	#$res = curl_get('http://localhost:8000/metrics');
	#$res = curl_get('http://172.22.10.150:8000/metrics');

	$metrics = preg_split("/\r\n|\n|\r/", $res);

	foreach ($metrics as $metric) {
		if (empty($metric)) continue;
		if ($metric[0] == '#') continue;
		#echo "$metric\n";
		#if (!preg_match('/^fazantix_stream_frames_([a-z]*)_total/', $metric, $matches)) continue;
		if (!preg_match('/^fazantix_stream_frames_([a-z]*)_total\{name="([a-z]*)"\} ([0-9]*)$/', $metric, $matches)) continue;

		$what = $matches[1];
		$stream = $matches[2];
		$count = $matches[3];

		if (!isset($logged[$stream])) $logged[$stream] = array();
		if (!isset($logged[$stream][$what])) $logged[$stream][$what] = 0;

		$diff = ($count - $logged[$stream][$what]);
		if ($diff < 30 && $diff != 0) {
			echo "$stream $what diff ".($count - $logged[$stream][$what]). "\n";
		}
		$logged[$stream][$what] = $count;
		
	}
	foreach($logged as $st => $vals) {
		if ($st !== "camera" && $st !== "slides") continue;
		echo $st."\tread ".$vals["read"]." req +".($vals["requested"]-$vals["read"])." written +".($vals["written"]-$vals["read"])."\n";
	}
	echo "=========================\n";
	sleep(1);	
}

?>
