const byte led = 13;
bool state = 1;

void setup() {
  // put your setup code here, to run once:
  Serial.begin(9600);
  pinMode(led, OUTPUT);
}

void loop() {
  // put your main code here, to run repeatedly:
  String buf;
  if(Serial.available()){
    buf = Serial.readString();
    if(buf[0]=='A' && buf[1]=='T' && buf[2]=='\r' && buf[3]=='\n'){
      Serial.println("OK");  
      digitalWrite(led, 1);
    }else{
      Serial.print(buf);  
    }
  }else{
//    Serial.println("Waiting");    
//    delay(500);
  }
}
