<?php
date_default_timezone_set('America/Chicago');
$timeZone = new \DateTimeZone('America/Chicago');
$dt = new \DateTimeImmutable();
$dt = $dt->setTimezone($timeZone);
$date = $dt->format('Y-m-d');
$time = $dt->format('Y-m-d H:i:s.v');


function updateTimeValue () {
    $dt = null;
    $date = null;
    $time = null;
    $timeZone = new \DateTimeZone('America/Chicago');
    $dt = new \DateTimeImmutable();
    $dt = $dt->setTimezone($timeZone);
    $date = $dt->format('Y-m-d');
    $time = $dt->format('Y-m-d H:i:s.v');
    return $time;
}

function strFilter ($string) {
    if ($string === "" || $string === " " || $string === "NULL" || empty($string) || is_null($string)) {
        return 1;
    } else {
        return 0;
    }
}

function numFilter ($string) {
    if (strFilter($string) === 0) {
        if (is_numeric($string) && $string > 0) {
            return 0;
        } else {
            return 1;
        }
    } else {
        return 1;
    }
}

function arrFilter ($arr) {
    if (is_array($arr)) {
        if (empty($arr)) {
            return 1;
        } else {
            return 0;
        }
    } else {
        return 1;
    }
}


class scriptTimer {
    private $arrMarker;
    private $arrMarkerObj;
    private $markerIt;
    private $arrTime;
    private $arrTimeObj;
    private $timeIt;
    private $curTime;

    private function refreshTime() {
        $this->curTime = hrtime(true);
    }

    private function clearTime() {
        $this->arrTime = null;
        $this->arrTimeObj = null;
        $this->timeIt = null;
    }

    private function clearMarker() {
        $this->arrMarker = null;
        $this->arrMarkerObj = null;
        $this->markerIt = null;
    }

    public function start() {
        $this->clearTime();
        $this->refreshTime();
        $this->arrTime = array();
        $this->arrTimeObj = new ArrayObject($this->arrTime);
        $this->timeIt = $this->arrTimeObj->getIterator();
        $this->arrTimeObj->append($this->curTime);
    }

    public function startMarker() {
        $this->clearMarker();
        $this->refreshTime();
        $this->arrMarker = array();
        $this->arrMarkerObj = new ArrayObject($this->arrMarker);
        $this->markerIt = $this->arrMarkerObj->getIterator();
        $this->arrMarkerObj->append($this->curTime);
    }

    public function endMarker() {
        $this->refreshTime();
        $this->arrMarkerObj->append($this->curTime);
        $count = $this->markerIt->count() - 1;
        $this->markerIt->seek($count);
        $end = $this->markerIt->current();
        $this->markerIt->rewind();
        $start = $this->markerIt->current();
        $execTime = round(($end - $start) / 1e9, 4);
        return $execTime;
        $this->clearMarker();
    }

    public function end() {
        $this->refreshTime();
        $this->arrTimeObj->append($this->curTime);
        $count = $this->timeIt->count() - 1;
        $this->timeIt->seek($count);
        $end = $this->timeIt->current();
        $this->timeIt->rewind();
        $start = $this->timeIt->current();
        $execTime = round(($end - $start) / 1e9, 4);
        return $execTime;
        $this->clearTime();
    }
}

class PostgreSQLConn {
    private static $user = "uitweb";
    private static $pass = "UIT_WEB_SVC_PASSWD";
    private static $host = "localhost";
    private static $dbName = "uitdb";
    private static $options = array(PDO::ATTR_PERSISTENT => true, PDO::ATTR_ERRMODE => true, PDO::ERRMODE_EXCEPTION => true);
    public function dbObj() {
        return new PDO("pgsql:host=" . self::$host . ";port=5432;dbname=" . self::$dbName . ";", self::$user, self::$pass, self::$options);
    }
}

class dbPSQL {
  private $sql;
  private $arr;
  private $pdo;
  private $dbPSQL;
  private $rowCount;

  public function __construct() {
    if (!isset($this->dbPSQL) || is_null($this->dbPSQL)) {
      $this->dbPSQL = new PostgreSQLConn();
    }
    $this->pdo = $this->dbPSQL->dbObj();
    $this->sql = "";
    $this->arr = array();
    $this->rowCount = 0;
  }


  public function check_tables_cols (string $table, string $cols) {
    if (strFilter($table) === 1 || strFilter($cols) === 1) {
      return 1;
      exit();
    }

    //ORDER OF THESE ENTRIES MATTER
    $acceptableTables = array();
    $stmt = $this->pdo->prepare("SELECT * FROM pg_catalog.pg_tables WHERE schemaname='public'");
    $stmt->execute();
    $tableArr = $stmt->fetchAll();
    foreach ($tableArr as $key => $value) {
      array_push($acceptableTables, $value["tablename"]);
    }
    unset($tableArr);
    unset($value);

    if (array_search($table, $acceptableTables) === false) {
      return 1;
      exit();
    }

    //ORDER OF THESE ENTRIES MATTER
    $acceptableCols = array();
    $stmt = $this->pdo->prepare("SELECT column_name FROM INFORMATION_SCHEMA.COLUMNS WHERE table_name = '" . $table . "'");
    $stmt->execute();
    $colsArr = $stmt->fetchAll();
    foreach ($colsArr as $key => $value) {
      array_push($acceptableCols, $value["column_name"]);
    }
    unset($colsArr);
    unset($value);

    if (array_search($cols, $acceptableCols) === false) {
      return 1;
      exit();
    }

    return 0;
  }


  public function select($sql) {
    $this->sql = $sql;
    $stmt = $this->pdo->prepare($sql);
    $stmt->execute();
    $this->arr = $stmt->fetchAll();
    $this->rowCount = $stmt->rowCount();
  }

  public function AssocSelect($sql) {
    $this->sql = $sql;
    $stmt = $this->pdo->prepare($sql);
    $stmt->execute();
    $this->arr = $stmt->fetchAll(PDO::FETCH_ASSOC);
    $this->rowCount = $stmt->rowCount();
  }

  public function Pselect($sql, $arr) {
    $this->sql = $sql;
    $stmt = $this->pdo->prepare($sql);
    $stmt->execute($arr);
    $this->arr = $stmt->fetchAll();
    $this->rowCount = $stmt->rowCount();
  }

  public function AssocPSelect($sql, $arr) {
    $this->sql = $sql;
    $stmt = $this->pdo->prepare($sql);
    $stmt->execute($arr);
    $this->arr = $stmt->fetchAll(PDO::FETCH_ASSOC);
    $this->rowCount = $stmt->rowCount();
  }

  public function get() {
    if(is_array($this->arr) && arrFilter($this->arr) === 0) {
      return $this->arr;
    } else {
      return "NULL";
    }
  }

  public function nested_get() {
    if(is_array($this->arr) && arrFilter($this->arr) === 0) {
      foreach ($this->arr as $nestedKey => $nestedValue) {
        $this->arr = $nestedValue;
      }
      return $this->arr;
    } else {
      return "NULL";
    }
  }

  public function get_rows() {
    return $this->rowCount;
  }


  // JOBSTATS table
  public function insertJob ($uuid) {
    if (strFilter($uuid) === 0) {
      $sql = "INSERT INTO jobstats (uuid) VALUES (:uuid)";
      $stmt = $this->pdo->prepare($sql);
      $stmt->bindParam(':uuid', $uuid, PDO::PARAM_STR);

      if (strFilter($stmt) === 0) {
        $stmt->execute();
      }
    }
  }

  public function updateJob ($key, $value, $uuid) {
    if (strFilter($key) === 0 && strFilter($uuid) === 0) {
      if ($this->check_tables_cols("jobstats", $key) === 0) {
        $sql = "UPDATE jobstats SET $key = :value WHERE uuid = :uuid";
        $stmt = $this->pdo->prepare($sql);

        if (is_bool($value) === true) {
          $stmt->bindParam(':uuid', $uuid, PDO::PARAM_STR); 
          $stmt->bindParam(':value', $value, PDO::PARAM_BOOL);
        } elseif (is_numeric($value) && numFilter($value) === 1) {
          $stmt->bindParam(':uuid', $uuid, PDO::PARAM_STR);
          $stmt->bindParam(':value', $value, PDO::PARAM_NULL);
        } elseif (strFilter($value) === 0) {
          $stmt->bindParam(':uuid', $uuid, PDO::PARAM_STR);
          $stmt->bindParam(':value', $value, PDO::PARAM_STR);
        } else {
          $stmt->bindParam(':uuid', $uuid, PDO::PARAM_STR);
          $stmt->bindParam('value', $value, PDO::PARAM_NULL);
        }

        if (strFilter($stmt) === 0) {
          $stmt->execute();
        }
      }
    }
  }


  // SERVERSTATS table
  public function insertSS ($date) {
    if (strFilter($date) === 0) {
      $sql = "INSERT INTO serverstats (date) VALUES (:date)";
      $stmt = $this->pdo->prepare($sql);
      $stmt->bindParam(':date', $date, PDO::PARAM_STR);

      if (strFilter($stmt) === 0) {
        $stmt->execute();
      }
    }
  }

  public function updateSS ($key, $value, $date) {
    if (strFilter($key) === 0 && strFilter($date) === 0) {
      if ($this->check_tables_cols("serverstats", $key) === 0) {
        $sql = "UPDATE serverstats SET $key = :value WHERE date = :date";
        $stmt = $this->pdo->prepare($sql);

        if (is_bool($value) === true) {
          $stmt->bindParam(':date', $date, PDO::PARAM_STR);
          $stmt->bindParam(':value', $value, PDO::PARAM_BOOL);
        } elseif (is_numeric($value) && numFilter($value) === 1) {
          $stmt->bindParam(':date', $date, PDO::PARAM_STR);
          $stmt->bindParam(':value', $value, PDO::PARAM_NULL);
        } elseif (strFilter($value) === 0) {
          $stmt->bindParam(':date', $date, PDO::PARAM_STR);
          $stmt->bindParam(':value', $value, PDO::PARAM_STR);
        } else {
          $stmt->bindParam(':date', $date, PDO::PARAM_STR);
          $stmt->bindParam(':value', $value, PDO::PARAM_NULL);
        }

        if (strFilter($stmt) === 0) {
          $stmt->execute();
        }
      }
    }
  }
    

  // Location table
  public function insertLocation ($time) {
    if (strFilter($time) === 0) {
      $sql = "INSERT INTO locations (time) VALUES (:time)";
      $stmt = $this->pdo->prepare($sql);

      $stmt->bindParam(':time', $time, PDO::PARAM_STR);

      if (strFilter($stmt) === 0) {
        $stmt->execute();
      }
    }
  }

  public function updateLocation ($key, $value, $time) {
    if (strFilter($key) === 0 && strFilter($time) === 0) {
      if ($this->check_tables_cols("locations", $key) === 0) {
        $sql = "UPDATE locations SET $key = :value WHERE time = :time";
        $stmt = $this->pdo->prepare($sql);

        if (is_bool($value) === true) {
          $stmt->bindParam(':time', $time, PDO::PARAM_STR); 
          $stmt->bindParam(':value', $value, PDO::PARAM_BOOL);
        } elseif (strFilter($value) === 0) {
          $stmt->bindParam(':time', $time, PDO::PARAM_STR);
          $stmt->bindParam(':value', $value, PDO::PARAM_STR);
        } else {
          $stmt->bindParam(':time', $time, PDO::PARAM_STR);
          $stmt->bindParam(':value', $value, PDO::PARAM_NULL);
        }

        if (strFilter($stmt) === 0) {
          $stmt->execute();
        }
      }
    }
  }
    

  // Remote table
  public function insertRemote ($tagNum) {
    if (strFilter($tagNum) === 0) {
      $sql = "INSERT INTO remote (tagnumber) VALUES (:tagNum)";
      $stmt = $this->pdo->prepare($sql);

      if (strFilter($tagNum) === 0) {
        $stmt->bindParam(':tagNum', $tagNum, PDO::PARAM_INT);
      } else {
        exit();
      }

      if (strFilter($stmt) === 0) {
        $stmt->execute();
      }
    }
  }

  public function updateRemote ($tagNum, $key, $value) {
    if (strFilter($tagNum) === 0 && strFilter($key) === 0) {
      if ($this->check_tables_cols("remote", $key) === 0) {
        $sql = "UPDATE remote SET $key = :value WHERE tagnumber = :tagNum";
        $stmt = $this->pdo->prepare($sql);

        if (is_bool($value) === true) {
          $stmt->bindParam(':tagNum', $tagNum, PDO::PARAM_INT);
          $stmt->bindParam(':value', $value, PDO::PARAM_BOOL);
        } elseif (strFilter($value) === 0) {
          $stmt->bindParam(':tagNum', $tagNum, PDO::PARAM_INT);
          $stmt->bindParam(':value', $value, PDO::PARAM_STR);
        } else {
          $stmt->bindParam(':tagNum', $tagNum, PDO::PARAM_INT);
          $stmt->bindParam(':value', $value, PDO::PARAM_NULL);
        }

        if (strFilter($stmt) == 0) {
          $stmt->execute();
        }
      }
    }
  }

    
  // SYSTEM_DATA table
  public function insertSystemData ($tagNum) {
    if (strFilter($tagNum) === 0) {
      $sql = "INSERT INTO system_data (tagnumber) VALUES (:tagNum)";
      $stmt = $this->pdo->prepare($sql);

      if (strFilter($tagNum) === 0) {
        $stmt->bindParam(':tagNum', $tagNum, PDO::PARAM_INT);
      } else {
        exit();
      }

      if (strFilter($stmt) === 0) {
        $stmt->execute();
      }
    }
  }

  public function updateSystemData ($tagNum, $key, $value) {
    if (strFilter($tagNum) === 0 && strFilter($key) === 0) {
      if ($this->check_tables_cols("system_data", $key) === 0) {
        $sql = "UPDATE system_data SET $key = :value WHERE tagnumber = :tagNum";
        $stmt = $this->pdo->prepare($sql);


        if (is_bool($value) === true) {
          $stmt->bindParam(':tagNum', $tagNum, PDO::PARAM_INT);
          $stmt->bindParam(':value', $value, PDO::PARAM_BOOL);
        } elseif (is_numeric($value) && numFilter($value) === 1) {
          $stmt->bindParam(':tagNum', $tagNum, PDO::PARAM_INT);
          $stmt->bindParam(':value', $value, PDO::PARAM_NULL);
        } elseif (strFilter($value) == 0) {
          $stmt->bindParam(':tagNum', $tagNum, PDO::PARAM_INT);
          $stmt->bindParam(':value', $value, PDO::PARAM_STR);
        } else {
          $stmt->bindParam(':tagNum', $tagNum, PDO::PARAM_INT);
          $stmt->bindParam(':value', $value, PDO::PARAM_NULL);
        }

        if (strFilter($stmt) == 0) {
          $stmt->execute();
        }

        $dt = new DateTimeImmutable();
        $time = $dt->format('Y-m-d H:i:s.v');

        $sql = "UPDATE system_data SET time = :clienttime WHERE tagnumber = :tagNum";
        $stmt = $this->pdo->prepare($sql);

        if (strFilter($time) == 0) {
          $stmt->bindParam(':tagNum', $tagNum, PDO::PARAM_INT);
          $stmt->bindParam(':clienttime', $time, PDO::PARAM_STR);
        } else {
          $stmt->bindParam(':tagNum', $tagNum, PDO::PARAM_INT);
          $stmt->bindParam(':clienttime', $time, PDO::PARAM_NULL);
        }

        if (strFilter($stmt) == 0) {
          $stmt->execute();
        }
      }
    }
  }


  // TODO/NOTES table
  public function insertToDo ($time) {
    if (strFilter($time) === 0) {
      $sql = "INSERT INTO notes (time) VALUES (:time)";
      $stmt = $this->pdo->prepare($sql);

      $stmt->bindParam(':time', $time, PDO::PARAM_STR);

      if (strFilter($stmt) === 0) {
        $stmt->execute();
      }
    }
  }

  public function updateToDo ($key, $value, $time) {
    if (strFilter($key) === 0 && strFilter($time) === 0) {
      if ($this->check_tables_cols("notes", $key) === 0) {
        $sql = "UPDATE notes SET $key = :value WHERE time = :time";
        $stmt = $this->pdo->prepare($sql);
        
        if (is_bool($value) === true) {
          $stmt->bindParam(':time', $time, PDO::PARAM_STR);
          $stmt->bindParam(':value', $value, PDO::PARAM_BOOL);
        } elseif (strFilter($value) === 0) {
          $stmt->bindParam(':time', $time, PDO::PARAM_STR);
          $stmt->bindParam(':value', $value, PDO::PARAM_STR);
        } else {
          $stmt->bindParam(':time', $time, PDO::PARAM_STR);
          $stmt->bindParam(':value', $value, PDO::PARAM_NULL);
        }

        if (strFilter($stmt) === 0) {
          $stmt->execute();
        }
      }
    }
  }


  // CHECKOUT table
  public function insertCheckout ($time) {
    if (strFilter($time) == 0) {
      $sql = "INSERT INTO checkouts (time) VALUES (:time)";
      $stmt = $this->pdo->prepare($sql);

      $stmt->bindParam(':time', $time, PDO::PARAM_STR);

      if (strFilter($stmt) === 0) {
        $stmt->execute();
      }
    }
  }

  public function updateCheckout ($key, $value, $time) {
    if (strFilter($key) === 0 && strFilter($time) === 0) {
      if ($this->check_tables_cols("checkouts", $key) === 0) {
        $sql = "UPDATE checkouts SET $key = :value WHERE time = :time";
        $stmt = $this->pdo->prepare($sql);

        if (is_bool($value) === true) {
          $stmt->bindParam(':time', $time, PDO::PARAM_STR);
          $stmt->bindParam(':value', $value, PDO::PARAM_BOOL);
        } elseif (strFilter($value) === 0) {
          $stmt->bindParam(':time', $time, PDO::PARAM_STR);
          $stmt->bindParam(':value', $value, PDO::PARAM_STR);
        } else {
          $stmt->bindParam(':time', $time, PDO::PARAM_STR);
          $stmt->bindParam(':value', $value, PDO::PARAM_NULL);
        }

        if (strFilter($stmt) === 0) {
          $stmt->execute();
        }
      }
    }
  }


  //CLIENT_HEALTH table
  public function insertClientHealth ($tagNum) {
    if (strFilter($tagNum) === 0) {
      $sql = "INSERT INTO client_health (tagnumber) VALUES (:tagNum)";
      $stmt = $this->pdo->prepare($sql);

      if (strFilter($tagNum) === 0) {
        $stmt->bindParam(':tagNum', $tagNum, PDO::PARAM_INT);
      } else {
        exit();
      }

      if (strFilter($stmt) === 0) {
        $stmt->execute();
      }
    }
  }

  public function updateClientHealth ($tagNum, $key, $value) {
    if (strFilter($tagNum) === 0 && strFilter($key) === 0) {
      if ($this->check_tables_cols("client_health", $key) === 0) {
        $sql = "UPDATE client_health SET $key = :value WHERE tagnumber = :tagNum";
        $stmt = $this->pdo->prepare($sql);
        
        if (is_bool($value) === true) {
          $stmt->bindParam(':tagNum', $tagNum, PDO::PARAM_INT);
          $stmt->bindParam(':value', $value, PDO::PARAM_BOOL);
        } elseif (strFilter($value) === 0) {
          $stmt->bindParam(':tagNum', $tagNum, PDO::PARAM_INT);
          $stmt->bindParam(':value', $value, PDO::PARAM_STR);
        } else {
          $stmt->bindParam(':tagNum', $tagNum, PDO::PARAM_INT);
          $stmt->bindParam(':value', $value, PDO::PARAM_NULL);
        }

        if (strFilter($stmt) == 0) {
          $stmt->execute();
        }
      }
    }
  }


  //CLIENT_IMAGES table
  public function insertImage ($uuid, $time, $tagnumber) {
    if (strFilter($uuid) === 0 && strFilter($time) === 0 && strFilter($tagnumber) === 0) {
      $sql = "INSERT INTO client_images (uuid, time, tagnumber) VALUES (:uuid, :time, :tagnumber)";
      $stmt = $this->pdo->prepare($sql);

      $stmt->bindParam(':uuid', $uuid, PDO::PARAM_STR);
      $stmt->bindParam(':time', $time, PDO::PARAM_STR);
      $stmt->bindParam(':tagnumber', $tagnumber, PDO::PARAM_INT);
    } else {
      exit();
    }

    if (strFilter($stmt) === 0) {
      $stmt->execute();
    }
  }

  public function updateImage ($key, $value, $uuid) {
    if (strFilter($uuid) === 0 && strFilter($key) === 0) {
      if ($this->check_tables_cols("client_images", $key) === 0) {
        $sql = "UPDATE client_images SET $key = :value WHERE uuid = :uuid";
        $stmt = $this->pdo->prepare($sql);

        if (is_bool($value) === true) {
          $stmt->bindParam(':uuid', $uuid, PDO::PARAM_STR); 
          $stmt->bindParam(':value', $value, PDO::PARAM_BOOL);
        } elseif (strFilter($value) === 0) {
          $stmt->bindParam(':uuid', $uuid, PDO::PARAM_STR);
          $stmt->bindParam(':value', $value, PDO::PARAM_STR);
        } else {
          $stmt->bindParam(':uuid', $uuid, PDO::PARAM_STR);
          $stmt->bindParam(':value', $value, PDO::PARAM_NULL);
        }

        if (strFilter($stmt) == 0) {
          $stmt->execute();
        }
      }
    }
  }

  public function deleteImage ($uuid, $time, $tagnumber) {
    if (strFilter($uuid) === 0 && strFilter($time) === 0 && strFilter($tagnumber) === 0) {
      $sql = "DELETE FROM client_images WHERE uuid = :uuid AND time = :time AND tagnumber = :tagnumber";
      $stmt = $this->pdo->prepare($sql);

      $stmt->bindParam(':uuid', $uuid, PDO::PARAM_STR);
      $stmt->bindParam(':time', $time, PDO::PARAM_STR);
      $stmt->bindParam(':tagnumber', $tagnumber, PDO::PARAM_STR);
    } else {
      exit();
    }

    if (strFilter($stmt) === 0) {
      $stmt->execute();
    }
  }


  //LIVE_IMAGES table
  public function insertLiveImages ($tagnumber) {
    if (strFilter($tagnumber) === 0) {
      $sql = "INSERT INTO live_images (tagnumber) VALUES (:tagnumber)";
      $stmt = $this->pdo->prepare($sql);

      $stmt->bindParam(':tagnumber', $tagnumber, PDO::PARAM_INT);
    } else {
      exit();
    }

    if (strFilter($stmt) === 0) {
      $stmt->execute();
    }
  }

  public function updateLiveImages ($tagnumber, $key, $value) {
    if (strFilter($tagnumber) === 0 && strFilter($key) === 0) {
      if ($this->check_tables_cols("live_images", $key) === 0) {
        $sql = "UPDATE live_images SET $key = :value WHERE tagnumber = :tagnumber";
        $stmt = $this->pdo->prepare($sql);

        if (is_bool($value) === true) {
          $stmt->bindParam(':tagnumber', $tagnumber, PDO::PARAM_STR);
          $stmt->bindParam(':value', $value, PDO::PARAM_BOOL);
        } elseif (strFilter($value) === 0) {
          $stmt->bindParam(':tagnumber', $tagnumber, PDO::PARAM_STR);
          $stmt->bindParam(':value', $value, PDO::PARAM_STR);
        } else {
          $stmt->bindParam(':tagnumber', $tagnumber, PDO::PARAM_STR);
          $stmt->bindParam(':value', $value, PDO::PARAM_NULL);
        }

        if (strFilter($stmt) == 0) {
          $stmt->execute();
        }
      }
    }
  }

}


function removeUrlVar ($url, $varName) {
  list($urlPart, $queryPart) = array_pad(explode('?', $url), 2, '');
  parse_str($queryPart, $queryVars);
  unset($queryVars[$varName]);
  $newUrl = http_build_query($queryVars);
  return $urlPart . "?" . $newUrl;
}

function addUrlVar ($url, $varName, $varValue) {
  list($urlPart, $queryPart) = array_pad(explode('?', $url), 2, '');
  parse_str($queryPart, $queryVars);
  unset($queryVars[$varName]);
  $queryVars[$varName] = $varValue;
  $newUrl = http_build_query($queryVars);
  return $urlPart . "?" . $newUrl;
}

function getUrlVar ($url) {
  list($urlPart, $queryPart) = array_pad(explode('?', $url), 2, '');
  parse_str($queryPart, $queryVars);
  $newUrl = http_build_query($queryVars);
  return $urlPart . "?" . $newUrl;
}

?>