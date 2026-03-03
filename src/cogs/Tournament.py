import discord
from discord.ext import commands
import asyncio
import random

from src.database.balance import get_balance, update_balance
from src.database.pets import get_active_pet, update_pet
from src.utils.embed_utils import generate_hp_bar


class Tournament(commands.Cog):
    def __init__(self, bot):
        self.bot = bot
        self.tournaments = {}

    @commands.group(name='tournoi', aliases=['tournament', 'tourney'], invoke_without_command=True)
    async def tournoi(self, ctx):
        """Gestion des tournois de familiers."""
        embed = discord.Embed(title="🏆 Tournois de Familiers", color=discord.Color.gold())
        embed.description = (
            "**Commandes disponibles :**\n"
            "`!tournoi create <mise>` : Crée un tournoi.\n"
            "`!tournoi join` : Rejoins le tournoi en payant la mise.\n"
            "`!tournoi start` : Lance le tournoi (Créateur uniquement).\n"
            "`!tournoi cancel` : Annule et rembourse tout le monde."
        )
        await ctx.send(embed=embed)

    @tournoi.command(name='create')
    async def t_create(self, ctx, fee: int):
        """Créer un nouveau tournoi (Mise requise)."""
        if fee < 0:
            return await ctx.send("❌ La mise ne peut pas être négative.")
        guild_id = ctx.guild.id
        if guild_id in self.tournaments:
            return await ctx.send("❌ Un tournoi est déjà en cours de préparation sur ce serveur !")
        if get_balance(ctx.author.id) < fee:
            return await ctx.send(f"❌ Tu n'as pas les {fee}$ nécessaires pour créer ce tournoi.")
        pet = get_active_pet(ctx.author.id)
        if not pet:
            return await ctx.send("❌ Tu dois avoir un familier actif pour créer un tournoi.")
        if fee > 0:
            update_balance(ctx.author.id, -fee)
        self.tournaments[guild_id] = {
            "creator": ctx.author,
            "fee": fee,
            "players": [ctx.author],
            "started": False
        }
        embed = discord.Embed(title="🏆 NOUVEAU TOURNOI !", color=discord.Color.gold())
        embed.description = (
            f"**{ctx.author.display_name}** organise un Grand Tournoi de Familiers !\n\n"
            f"💰 **Frais d'inscription :** {fee} $\n"
            f"🎁 **Cash Prize Actuel :** {fee} $\n\n"
            f"Tapez `!tournoi join` pour vous inscrire !"
        )
        return await ctx.send(embed=embed)

    @tournoi.command(name='join')
    async def t_join(self, ctx):
        """Rejoindre le tournoi en cours."""
        guild_id = ctx.guild.id
        if guild_id not in self.tournaments:
            return await ctx.send("❌ Aucun tournoi en préparation. Utilise `!tournoi create <mise>`.")
        t = self.tournaments[guild_id]
        if t["started"]:
            return await ctx.send("❌ Le tournoi a déjà commencé !")
        if ctx.author in t["players"]:
            return await ctx.send("❌ Tu es déjà inscrit !")
        if get_balance(ctx.author.id) < t["fee"]:
            return await ctx.send(f"❌ Il te faut {t['fee']} $ pour t'inscrire.")
        pet = get_active_pet(ctx.author.id)
        if not pet:
            return await ctx.send("❌ Tu dois équiper un familier d'abord (`!equip`).")
        if t["fee"] > 0:
            update_balance(ctx.author.id, -t["fee"])
        t["players"].append(ctx.author)
        cash_prize = len(t["players"]) * t["fee"]
        await ctx.send(
            f"✅ **{ctx.author.display_name}** rejoint le tournoi avec **{pet.nickname}** ! (Cash Prize: {cash_prize} $)")
        return None

    @tournoi.command(name='start')
    async def t_start(self, ctx):
        """Lancer le tournoi (Créateur ou Admin)."""
        guild_id = ctx.guild.id
        if guild_id not in self.tournaments:
            return await ctx.send("❌ Aucun tournoi actif.")
        t = self.tournaments[guild_id]
        if ctx.author != t["creator"] and not ctx.author.guild_permissions.administrator:
            return await ctx.send("❌ Seul le créateur du tournoi peut le lancer.")
        if len(t["players"]) < 2:
            return await ctx.send("❌ Il faut au moins 2 joueurs pour lancer le tournoi !")
        t["started"] = True
        players = t["players"]
        random.shuffle(players)

        cash_prize = len(players) * t["fee"]

        embed = discord.Embed(title="⚔️ LE TOURNOI COMMENCE ! ⚔️", color=discord.Color.red())
        embed.description = f"**{len(players)} Dresseurs** s'affrontent pour le Cash Prize de **{cash_prize} $** !\nPréparez-vous..."
        await ctx.send(embed=embed)
        await asyncio.sleep(3)

        round_num = 1
        consecutive_final_ties = 0
        co_champions = None

        while len(players) > 1:
            await ctx.send(f"🛡️ **--- ROUND {round_num} ---** 🛡️")
            next_round_players = []
            is_final = len(players) == 2
            for i in range(0, len(players), 2):
                if i + 1 < len(players):
                    p1 = players[i]
                    p2 = players[i + 1]
                    winners = await self.simulate_match(ctx, p1, p2)
                    if len(winners) == 2 and is_final:
                        consecutive_final_ties += 1
                        if consecutive_final_ties >= 3:
                            co_champions = winners
                            next_round_players = winners
                            break
                    else:
                        consecutive_final_ties = 0
                    next_round_players.extend(winners)
                    await asyncio.sleep(2)
                else:
                    lucky_player = players[i]
                    next_round_players.append(lucky_player)
                    await ctx.send(f"🍀 *{lucky_player.display_name} passe automatiquement au tour suivant !*")
            if co_champions:
                break
            players = next_round_players
            round_num += 1
            await asyncio.sleep(2)
        if co_champions:
            p1, p2 = co_champions
            half_prize = cash_prize // 2
            if half_prize > 0:
                update_balance(p1.id, half_prize)
                update_balance(p2.id, half_prize)
            embed = discord.Embed(title="🤝 ÉGALITÉ EN FINALE ! 🏆", color=discord.Color.gold())
            embed.description = (
                f"👑 Après 3 matchs nuls consécutifs, **{p1.mention}** et **{p2.mention}** sont déclarés Co-Champions !\n\n"
                f"💰 **Le Cash Prize est partagé : {half_prize} $ chacun !**"
            )
            await ctx.send(embed=embed)
        else:
            champion = players[0]
            champ_pet = get_active_pet(champion.id)
            if cash_prize > 0:
                update_balance(champion.id, cash_prize)
            embed = discord.Embed(title="🏆 CHAMPION DU TOURNOI ! 🏆", color=discord.Color.gold())
            embed.description = (
                f"👑 **{champion.mention}** remporte le tournoi avec {champ_pet.emoji} **{champ_pet.nickname}** !\n\n"
                f"💰 **Gain total : {cash_prize} $**"
            )
            await ctx.send(embed=embed)
        del self.tournaments[guild_id]
        return None

    async def simulate_match(self, ctx, user1, user2):
        pet1 = get_active_pet(user1.id)
        pet2 = get_active_pet(user2.id)
        pet1.heal_full()
        pet2.heal_full()
        turn = 1
        fighters = [pet1, pet2] if pet1.speed >= pet2.speed else [pet2, pet1]
        embed = discord.Embed(title="⚔️ ARÈNE DES FAMILIERS ⚔️", color=discord.Color.dark_theme())

        def update_embed_fields():
            embed.clear_fields()
            embed.add_field(
                name=f"{pet1.emoji} {pet1.nickname} (Niv {pet1.level})",
                value=f"Maître : {user1.display_name}\nPV : {generate_hp_bar(pet1.hp, pet1.max_hp)}\n`{int(pet1.hp)} / {pet1.max_hp}`",
                inline=True
            )
            embed.add_field(name="VS", value="⚡", inline=True)
            embed.add_field(
                name=f"{pet2.emoji} {pet2.nickname} (Niv {pet2.level})",
                value=f"Maître : {user2.display_name}\nPV : {generate_hp_bar(pet2.hp, pet2.max_hp)}\n`{int(pet2.hp)} / {pet2.max_hp}`",
                inline=True
            )
        update_embed_fields()
        embed.description = "🔥 *Le public retient son souffle...*"
        msg = await ctx.send(content=None, embed=embed, view=None)
        await asyncio.sleep(1)

        log = []
        while pet1.is_alive and pet2.is_alive and turn <= 15:
            for i in range(2):
                attacker = fighters[i]
                defender = fighters[1 - i]
                if attacker.is_alive:
                    action_text = attacker.attack(defender)
                    if len(log) > 5:
                        log.pop(0)
                    log.append(action_text)
                    update_embed_fields()
                    embed.description = "📜 **Journal de combat :**\n\n" + "\n".join(log)
                    await msg.edit(embed=embed)
                    await asyncio.sleep(0.5)
            turn += 1

        if pet1.is_alive and not pet2.is_alive:
            winners = [user1]
            win_pet, lose_pet = pet1, pet2
        elif pet2.is_alive and not pet1.is_alive:
            winners = [user2]
            win_pet, lose_pet = pet2, pet1
        else:
            winners = [user1, user2]
            win_pet, lose_pet = None, None

        if len(winners) == 1:
            winner = winners[0]
            diff_win, diff_lose = win_pet.update_elo(lose_pet, 1.0)
            embed.set_footer(text=f"🏆 VICTOIRE DE {winner.display_name.upper()} !")
        else:
            diff1, diff2 = pet1.update_elo(pet2, 0.5)
            d1_str = f"+{diff1}" if diff1 > 0 else str(diff1)
            d2_str = f"+{diff2}" if diff2 > 0 else str(diff2)
            embed.set_footer(text="🤝 MATCH NUL ! Limite de tours atteinte.")
            embed.description += "\n\n💨 *Les deux familiers s'effondrent de fatigue...*"
        pet1.heal_full()
        pet2.heal_full()
        update_pet(pet1)
        update_pet(pet2)
        await msg.edit(embed=embed)
        return winners

    @tournoi.command(name='cancel')
    async def t_cancel(self, ctx):
        """Annuler le tournoi et rembourser les joueurs."""
        guild_id = ctx.guild.id
        if guild_id not in self.tournaments:
            return await ctx.send("❌ Aucun tournoi actif.")
        t = self.tournaments[guild_id]
        if ctx.author != t["creator"] and not ctx.author.guild_permissions.administrator:
            return await ctx.send("❌ Seul le créateur peut annuler.")
        if t["started"]:
            return await ctx.send("❌ Trop tard, les combats ont commencé !")
        if t["fee"] > 0:
            for player in t["players"]:
                update_balance(player.id, t["fee"])
        del self.tournaments[guild_id]
        return await ctx.send("🛑 **Le tournoi a été annulé.** Tous les participants ont été remboursés.")


async def setup(bot):
    await bot.add_cog(Tournament(bot))