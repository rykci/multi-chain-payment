<template>
  <div class="metamaskLogin login">
    <el-alert type="warning" effect="dark" center show-icon v-if="metaAddress&&networkID!=80001">
        <div slot="title">{{$t('fs3Login.toptip_01')}} {{metaNetworkInfo.name}} {{$t('fs3Login.toptip_02')}} <span style="text-decoration: underline;">{{$t('fs3Login.toptip_Network')}}</span>.</div>
    </el-alert>
    <div class="metamask">
      <div class="titleCont">{{$t('fs3Login.title')}}</div>
      <el-row>
          <el-col :span="24">
            <img src="@/assets/images/metamask.png" alt="">
            <span>{{$t('fs3Login.MetaMask')}}</span>
          </el-col>
      </el-row>
      <div class="login_footer">
          <el-button type="primary" @click="signFun">{{$t('fs3.Connect_Wallet')}}</el-button>
          <p v-if="metaNetworkInfo.center_fail">{{$t('fs3Login.MetaMask_tip')}}</p>
      </div>
    </div>
  </div>
</template>

<script>
  import NCWeb3 from "@/utils/web3";
  export default {
    name: 'login',
    data() {
      return {
        fromEnter: ''
      }
    },
    methods: {
      signFun(){
          let _this = this
          if(!_this.metaAddress || _this.metaAddress == 'undefined'){
              NCWeb3.Init(addr=>{
                  _this.$nextTick(() => {
                      _this.$store.dispatch('setMetaAddress', addr)
                      _this.$emit("getMetamaskLogin", true)
                  })
              })
              return false
          }
      },
      walletInfo() {
          let _this = this
          if(!_this.metaAddress || _this.metaAddress == 'undefined'){
              return false
          }
      },
      // 是否已登录
      isLogin() {
        var _this = this
        if (_this.metaAddress && _this.networkID == 80001) {
          _this.$router.push({ path: '/upload_file' })
        }
      }
    },
    computed: {
      metaAddress() {
          return this.$store.getters.metaAddress
      },
      networkID() {
          return this.$store.getters.networkID
      },
      metaNetworkInfo() {
          return this.$store.getters.metaNetworkInfo?JSON.parse(JSON.stringify(this.$store.getters.metaNetworkInfo)):{}
      }
    },
    mounted() {
      this.isLogin()
      this.fromEnter = this.$route.query.redirect
    }
  }
</script>

<style rel="stylesheet/scss" lang="scss" scopte>
.metamaskLogin{
  display: flex;
  justify-content: center;
  align-items: center;
  position: absolute;
  width: 100%;
  height: 100%;
  .el-alert{
      position: absolute;
      left: 0;
      top: 0;
      .el-alert__content{
          display: flex;
          align-items: center;
      }
  }
  .metamask{
    width: 90%;
    max-width: 550px;
    padding: 3%;
    background: #fff;
    border-radius: 3px;
    .titleCont{
      font-weight: 600;
      font-size: 0.24rem;
      color: #333;
      @media screen and (max-width:600px){
          font-size: 16px;
      }
    }
    .el-row{
      border-radius: 0.08rem;
      margin: 0.25rem 0px;
      border: 1px solid rgb(229, 232, 235);
      text-align: center;
      position: static;
      transition: all 0.3s ease 0s;
      overflow: hidden;
      &:hover{
        // background: rgba(240, 185, 11, 0.1);
        // border: 1px solid rgb(240, 185, 11);
        box-shadow: 0 0 6px rgba(0,0,0,0.1);
      }
      .el-col{
        display: flex;
        -webkit-box-pack: justify;
        justify-content: flex-start;
        -webkit-box-align: center;
        align-items: center;
        background: #fff;
        padding: 0.16rem;
        transition: all 0.3s ease 0s;
        cursor: pointer;
        img{
          display: block;
          height: 0.24rem;
          margin: 0 0.15rem 0 0;
          order: 1;
          @media screen and (max-width:600px){
              height: 24px;
          }
        }
        span{
          order: 2;
          font-size: 0.14rem;
          @media screen and (max-width:600px){
              font-size: 14px;
          }
        }
        &:hover{
          // background: #eee;
        }
      }
    }
    .login_footer{
      .el-button{
        display: block;
        width: 100%;
        font-size: 0.14rem;
        height: 0.4rem;
        padding: 0;
        background: #5c3cd3;
        color: #fff;
        border-radius: 0.08rem;
        @media screen and (max-width:600px){
            font-size: 14px;
        }
        &:hover{
          background: #4326ab;
        }
      }
      p{
          font-size: 0.13rem;
          line-height: 1.5;
          color: red;
          margin: 0.1rem 0 0;
          @media screen and (max-width:600px){
              font-size: 12px;
          }
      }
    }
  }
}
</style>
