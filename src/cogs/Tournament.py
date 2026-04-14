import discord
from discord.ext import commands
import asyncio
import random

from src.database.balance import get_balance, update_balance
from src.database.pets import get_active_pet, update_pet
from src.utils.embed_utils import generate_hp_bar
from src.utils.battle import simulate_battle
from src.utils.i18n import t
from src.database.settings import get_language


class Tournament(commands.Cog):
    def __init__(self, bot):
        self.bot = bot
        self.tournaments = {}

    @commands.group(name='tournoi', aliases=['tournament', 'tourney'], invoke_without_command=True)
    async def tournoi(self, ctx):
        """Gestion des tournois de familiers."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        embed = discord.Embed(title=t("tournament.title", lang), color=discord.Color.gold())
        embed.description = t("tournament.desc", lang)
        await ctx.send(embed=embed)

    @tournoi.command(name='create')
    async def t_create(self, ctx, fee: int):
        """Créer un nouveau tournoi (Mise requise)."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        if fee < 0:
            return await ctx.send(t("tournament.invalid_fee", lang))
        guild_id = ctx.guild.id
        if guild_id in self.tournaments:
            return await ctx.send(t("tournament.already_prep", lang))
        if get_balance(ctx.author.id) < fee:
            return await ctx.send(t("tournament.no_money", lang, fee=fee))
        pet = get_active_pet(ctx.author.id)
        if not pet:
            return await ctx.send(t("tournament.no_pet", lang))
        if fee > 0:
            update_balance(ctx.author.id, -fee)
        self.tournaments[guild_id] = {
            "creator": ctx.author,
            "fee": fee,
            "players": [ctx.author],
            "started": False
        }
        embed = discord.Embed(title=t("tournament.new_title", lang), color=discord.Color.gold())
        embed.description = t("tournament.new_desc", lang, user=ctx.author.display_name, fee=fee)
        return await ctx.send(embed=embed)

    @tournoi.command(name='join')
    async def t_join(self, ctx):
        """Rejoindre le tournoi en cours."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        guild_id = ctx.guild.id
        if guild_id not in self.tournaments:
            return await ctx.send(t("tournament.not_found", lang))
        tourney = self.tournaments[guild_id]
        if tourney["started"]:
            return await ctx.send(t("tournament.started_error", lang))
        if ctx.author in tourney["players"]:
            return await ctx.send(t("tournament.already_joined", lang))
        if get_balance(ctx.author.id) < tourney["fee"]:
            return await ctx.send(t("tournament.join_no_money", lang, fee=tourney['fee']))
        pet = get_active_pet(ctx.author.id)
        if not pet:
            return await ctx.send(t("tournament.join_no_pet", lang))
        if tourney["fee"] > 0:
            update_balance(ctx.author.id, -tourney["fee"])
        tourney["players"].append(ctx.author)
        cash_prize = len(tourney["players"]) * tourney["fee"]
        await ctx.send(t("tournament.joined_msg", lang, user=ctx.author.display_name, pet=pet.nickname, prize=cash_prize))
        return None

    @tournoi.command(name='start')
    async def t_start(self, ctx):
        """Lancer le tournoi (Créateur ou Admin)."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        guild_id = ctx.guild.id
        if guild_id not in self.tournaments:
            return await ctx.send(t("tournament.no_tournament", lang))
        tourney = self.tournaments[guild_id]
        if ctx.author != tourney["creator"] and not ctx.author.guild_permissions.administrator:
            return await ctx.send(t("tournament.only_creator_start", lang))
        if len(tourney["players"]) < 2:
            return await ctx.send(t("tournament.min_players", lang))
        tourney["started"] = True
        players = tourney["players"]
        random.shuffle(players)
        cash_prize = len(players) * tourney["fee"]

        embed = discord.Embed(title=t("tournament.started_title", lang), color=discord.Color.red())
        embed.description = t("tournament.started_desc", lang, count=len(players), prize=cash_prize)
        await ctx.send(embed=embed)
        await asyncio.sleep(3)

        round_num = 1
        consecutive_final_ties = 0
        co_champions = None

        while len(players) > 1:
            await ctx.send(t("tournament.round_title", lang, num=round_num))
            random.shuffle(players)
            next_round_players = []
            is_final = len(players) == 2
            for i in range(0, len(players), 2):
                if i + 1 < len(players):
                    p1, p2 = players[i], players[i + 1]
                    winners = await self.simulate_match(ctx, p1, p2, lang)
                    if len(winners) == 2 and is_final:
                        consecutive_final_ties += 1
                        if consecutive_final_ties >= 3:
                            co_champions = winners
                            next_round_players = winners
                            break
                    else: consecutive_final_ties = 0
                    next_round_players.extend(winners)
                    await asyncio.sleep(1)
                else:
                    lucky_player = players[i]
                    next_round_players.append(lucky_player)
                    await ctx.send(t("tournament.lucky_pass", lang, user=lucky_player.display_name))
            if co_champions: break
            players = next_round_players
            round_num += 1
            await asyncio.sleep(2)

        if co_champions:
            p1, p2 = co_champions
            half_prize = cash_prize // 2
            if half_prize > 0:
                update_balance(p1.id, half_prize)
                update_balance(p2.id, half_prize)
            embed = discord.Embed(title=t("tournament.final_tie_title", lang), color=discord.Color.gold())
            embed.description = t("tournament.final_tie_desc", lang, count=3, p1=p1.mention, p2=p2.mention, prize=half_prize)
            await ctx.send(embed=embed)
        else:
            champion = players[0]
            champ_pet = get_active_pet(champion.id)
            if cash_prize > 0:
                update_balance(champion.id, cash_prize)
            embed = discord.Embed(title=t("tournament.champion_title", lang), color=discord.Color.gold())
            embed.description = t("tournament.champion_desc", lang, user=champion.mention, emoji=champ_pet.emoji, pet=champ_pet.nickname, prize=cash_prize)
            await ctx.send(embed=embed)
            
        del self.tournaments[guild_id]
        return None

    async def simulate_match(self, ctx, user1, user2, lang):
        pet1, pet2 = get_active_pet(user1.id), get_active_pet(user2.id)
        pet1.heal_full(); pet2.heal_full()
        embed = discord.Embed(title=t("pets.battle.arena_title", lang), color=discord.Color.dark_theme())
        def update_embed_fields():
            embed.clear_fields()
            embed.add_field(name=f"{pet1.emoji} {pet1.nickname} (Niv {pet1.level})", value=t("tournament.master_label", lang, name=user1.display_name) + f"\nPV : {generate_hp_bar(pet1.hp, pet1.max_hp)}\n`{int(pet1.hp)} / {pet1.max_hp}`", inline=True)
            embed.add_field(name="VS", value="⚡", inline=True)
            embed.add_field(name=f"{pet2.emoji} {pet2.nickname} (Niv {pet2.level})", value=t("tournament.master_label", lang, name=user2.display_name) + f"\nPV : {generate_hp_bar(pet2.hp, pet2.max_hp)}\n`{int(pet2.hp)} / {pet2.max_hp}`", inline=True)
        update_embed_fields()
        embed.description = t("pets.battle.arena_intro", lang)
        msg = await ctx.send(embed=embed)
        await asyncio.sleep(1)
        await simulate_battle(pet1, pet2, msg, embed, update_embed_fields, sleep_time=0.2, send_messages=True, log_size=10, lang=lang)
        if pet1.is_alive and not pet2.is_alive: winners = [user1]; win_pet, lose_pet = pet1, pet2; result = 1.0
        elif pet2.is_alive and not pet1.is_alive: winners = [user2]; win_pet, lose_pet = pet2, pet1; result = 0.0
        else: winners = [user1, user2]; win_pet = pet1; lose_pet = pet2; result = 0.5
        if len(winners) == 1:
            winner = winners[0]
            win_pet.update_elo(lose_pet, 1.0)
            embed.set_footer(text=t("pets.battle.victory_footer", lang, name=winner.display_name.upper()))
        else:
            pet1.update_elo(pet2, 0.5)
            embed.set_footer(text=t("pets.battle.draw_footer", lang))
            embed.description += t("pets.battle.draw_msg", lang)
        pet1.heal_full(); pet2.heal_full()
        update_pet(pet1); update_pet(pet2)
        await msg.edit(embed=embed)
        return winners

    @tournoi.command(name='cancel')
    async def t_cancel(self, ctx):
        """Annuler le tournoi et rembourser les joueurs."""
        lang = get_language(ctx.guild.id if ctx.guild else None)
        guild_id = ctx.guild.id
        if guild_id not in self.tournaments:
            return await ctx.send(t("tournament.no_tournament", lang))
        tourney = self.tournaments[guild_id]
        if ctx.author != tourney["creator"] and not ctx.author.guild_permissions.administrator:
            return await ctx.send(t("tournament.cancel_error", lang))
        if tourney["started"]:
            return await ctx.send(t("tournament.cancel_too_late", lang))
        if tourney["fee"] > 0:
            for player in tourney["players"]:
                update_balance(player.id, tourney["fee"])
        del self.tournaments[guild_id]
        return await ctx.send(t("tournament.cancelled_msg", lang))


async def setup(bot):
    await bot.add_cog(Tournament(bot))